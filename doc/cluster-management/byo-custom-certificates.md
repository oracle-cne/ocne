# BYO Clusters with Custom Certificates

The "Bring Your Own" provider supports certificates supplied by the operator.
Set `customCertificates: true` to skip Kubernetes certificate generation by the
CLI and supply the credentials through Ignition instead.
The setting is supported only by the BYO provider and defaults to `false`.

This example creates certificate authorities, provisions credentials for each
node, and embeds them into Ignition using `extraIgnition`.

## Prerequisites

Use an Oracle Linux x86_64 host with libvirt and QEMU installed.  This example
uses the running `default` libvirt network on `192.168.122.0/24`, an active
directory storage pool named `images`, and an unused OCK boot image stored in
that pool as `boot.qcow2-1.33`.  Prepare the boot image as described in
[Bring Your Own Infrastructure](byo.md#creating-a-cluster-and-joining-nodes).
The image must not have been booted with Ignition already applied.

The cluster uses virtual IP `192.168.122.230`.  The two control planes use
`192.168.122.231` and `192.168.122.232`, and the worker uses `192.168.122.233`.
Use unused addresses and VM names.  Adjust the addresses together if the libvirt
network uses a different subnet.

The administration host needs Bash, OpenSSL, `ocne`, `kubectl`, `virsh`,
`qemu-img`, `jq`, and the Mike Farah implementation of `yq`.  All certificates
and keys in this example are generated with OpenSSL.  `kubectl config` only
assembles kubeconfig files from those credentials.

Run the commands in the same Bash session.  Create a private workspace and
cluster configuration.

```bash
umask 077
mkdir byo-custom-certificates
cd byo-custom-certificates
export WORK_DIR="$PWD"
mkdir -p pki/etcd

cat > byo.yaml <<'YAML'
name: byo-custom-certificates
provider: byo
kubernetesVersion: "1.33"
customCertificates: true
virtualIp: 192.168.122.230
providers:
  byo:
    networkInterface: enp1s0
    automaticTokenCreation: true
YAML

export CLUSTER_CONFIG="$PWD/byo.yaml"
export CLUSTER_NAME=$(yq -r .name "$CLUSTER_CONFIG")
export API_VIP=$(yq -r .virtualIp "$CLUSTER_CONFIG")
export KUBERNETES_SERVICE_IP=10.96.0.1
export CONTROL_PLANE_1_IP=192.168.122.231
export CONTROL_PLANE_2_IP=192.168.122.232
export WORKER_IP=192.168.122.233

export LIBVIRT_NETWORK=default
export LIBVIRT_POOL=images
export BOOT_VOLUME=boot.qcow2-1.33
export BOOT_VOLUME_PATH=$(sudo virsh vol-path --pool "$LIBVIRT_POOL" "$BOOT_VOLUME")
export POOL_PATH=$(dirname "$BOOT_VOLUME_PATH")
```

`KUBERNETES_SERVICE_IP` is the first usable address in the service subnet.
The value above uses the default service subnet, `10.96.0.0/12`.  Change it if
`byo.yaml` uses a different service subnet.

The generated Ignition files contain private keys.  Deliver them using the
protected provisioning method used for the cluster.

## Define the Virtual Machines

Define a helper based on the
[BYO libvirt test example](../../bats-tests/scenarios/libvirt/deployments/byo-3cp3worker/env.sh).
It reserves the node's DHCP address, creates its disk from the boot image,
attaches its Ignition configuration, and starts the VM.

```bash
create_domain() {
  local node=$1 address=$2 mac=$3

  echo "Reserving $address for $node"
  sudo virsh net-update "$LIBVIRT_NETWORK" add ip-dhcp-host \
    "<host mac='$mac' name='$node' ip='$address'/>" --live --config

  echo "Creating the disk and staging Ignition for $node"
  sudo qemu-img create -f qcow2 -F qcow2 -b "$BOOT_VOLUME_PATH" \
    "$POOL_PATH/$node.qcow2" 30G
  sudo install -m 0644 "$WORK_DIR/$node.ign" "$POOL_PATH/$node.ign"
  sudo virsh pool-refresh "$LIBVIRT_POOL"

  cat > "$WORK_DIR/$node.xml" <<XML
<domain type='kvm' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>$node</name>
  <memory unit='MiB'>4096</memory>
  <vcpu>2</vcpu>
  <os firmware='efi'>
    <type arch='x86_64' machine='q35'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features><acpi/><apic/><smm state='off'/></features>
  <cpu mode='host-passthrough'/>
  <on_reboot>restart</on_reboot>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='$LIBVIRT_POOL' volume='$node.qcow2'/>
      <target dev='sda' bus='scsi'/>
    </disk>
    <controller type='scsi' model='virtio-scsi'/>
    <interface type='network'>
      <mac address='$mac'/>
      <source network='$LIBVIRT_NETWORK'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <serial type='pty'><target port='0'/></serial>
    <console type='pty'><target type='serial' port='0'/></console>
    <rng model='virtio'><backend model='random'>/dev/urandom</backend></rng>
  </devices>
  <qemu:commandline>
    <qemu:arg value='-fw_cfg'/>
    <qemu:arg value='name=opt/com.coreos/config,file=$POOL_PATH/$node.ign'/>
  </qemu:commandline>
</domain>
XML

  echo "Defining and starting $node"
  sudo virsh define "$WORK_DIR/$node.xml"
  sudo virsh start "$node"
}
```

## Create the Certificate Authorities

Create the Kubernetes, etcd, and front-proxy authorities and the service account
signing keys.  These are shared by all control planes.  The example certificates
are valid for one year.

```bash
for authority in ca etcd/ca front-proxy-ca; do
  echo "Creating the $authority certificate authority"
  openssl req -x509 -newkey rsa:2048 -nodes -days 365 -sha256 \
    -subj "/CN=$CLUSTER_NAME-${authority//\//-}" \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign' \
    -keyout "pki/$authority.key" -out "pki/$authority.crt"
done

echo "Creating the shared service account signing keys"
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out pki/sa.key
openssl pkey -in pki/sa.key -pubout -out pki/sa.pub
```

To use existing authorities, place their certificates and keys at these same
paths instead of generating them.

## Provision Control Plane Credentials

Define helpers to sign a certificate with OpenSSL and assemble a kubeconfig
from a certificate and key.  The signing helper accepts an output path prefix,
a CA path prefix, a subject, an extended key usage, and optional SANs.

```bash
issue_certificate() {
  local output=$1 authority=$2 subject=$3 usage=$4 sans=${5:-}

  echo "Creating the private key and certificate request for $output"
  openssl req -new -newkey rsa:2048 -nodes -subj "$subject" \
    -keyout "$output.key" -out "$WORK_DIR/certificate.csr"
  echo "Signing $output with $authority"
  openssl x509 -req -in "$WORK_DIR/certificate.csr" \
    -CA "$authority.crt" -CAkey "$authority.key" -CAcreateserial \
    -days 365 -sha256 -out "$output.crt" \
    -extfile <(printf '%s\n' \
      'basicConstraints=critical,CA:FALSE' \
      'keyUsage=critical,digitalSignature,keyEncipherment' \
      "extendedKeyUsage=$usage" "${sans:+subjectAltName=$sans}")
}

write_kubeconfig() {
  local certificate=$1 identity=$2 destination=$3

  echo "Writing $destination for $identity"
  kubectl config --kubeconfig "$destination" set-cluster kubernetes \
    --server="https://$API_VIP:6443" \
    --certificate-authority="$WORK_DIR/pki/ca.crt" --embed-certs=true
  kubectl config --kubeconfig "$destination" set-credentials "$identity" \
    --client-certificate="$certificate.crt" \
    --client-key="$certificate.key" --embed-certs=true
  kubectl config --kubeconfig "$destination" set-context default \
    --cluster=kubernetes --user="$identity"
  kubectl config --kubeconfig "$destination" use-context default
}
```

Define a function that provisions a control plane.  Each node receives the
shared authorities and signing keys, its own server and client certificates,
and kubeconfigs for the administrator, controller manager, scheduler, and
kubelet.  The API server certificate includes the node address, API virtual IP,
and Kubernetes Service names and address.

```bash
prepare_control_plane() {
  local node=$1 address=$2
  local directory="$WORK_DIR/$node"
  local kubernetes="$directory/files/etc/kubernetes"
  local pki="$kubernetes/pki"
  local ca="$WORK_DIR/pki/ca"
  local etcd_ca="$WORK_DIR/pki/etcd/ca"
  local node_sans="DNS:$node,DNS:localhost,IP:$address,IP:127.0.0.1,IP:::1"
  local api_sans="$node_sans,IP:$API_VIP,IP:$KUBERNETES_SERVICE_IP"
  api_sans+=",DNS:kubernetes,DNS:kubernetes.default"
  api_sans+=",DNS:kubernetes.default.svc,DNS:kubernetes.default.svc.cluster.local"

  echo "Preparing credentials for control plane $node at $address"
  mkdir -p "$pki/etcd"
  cp "$ca.crt" "$ca.key" "$WORK_DIR/pki/front-proxy-ca.crt" \
    "$WORK_DIR/pki/front-proxy-ca.key" "$WORK_DIR/pki/sa.key" \
    "$WORK_DIR/pki/sa.pub" "$pki/"
  cp "$etcd_ca.crt" "$etcd_ca.key" "$pki/etcd/"
  printf '%s\n' "$node" > "$directory/files/etc/hostname"

  issue_certificate "$pki/apiserver" "$ca" '/CN=kube-apiserver' \
    serverAuth "$api_sans"
  issue_certificate "$pki/apiserver-kubelet-client" "$ca" \
    '/CN=kube-apiserver-kubelet-client/O=system:masters' clientAuth
  issue_certificate "$pki/front-proxy-client" "$WORK_DIR/pki/front-proxy-ca" \
    '/CN=front-proxy-client' clientAuth
  issue_certificate "$pki/apiserver-etcd-client" "$etcd_ca" \
    '/CN=kube-apiserver-etcd-client' clientAuth
  issue_certificate "$pki/etcd/server" "$etcd_ca" "/CN=$node" \
    serverAuth,clientAuth "$node_sans"
  issue_certificate "$pki/etcd/peer" "$etcd_ca" "/CN=$node" \
    serverAuth,clientAuth "$node_sans"
  issue_certificate "$pki/etcd/healthcheck-client" "$etcd_ca" \
    '/CN=kube-etcd-healthcheck-client' clientAuth

  issue_certificate "$pki/admin" "$ca" \
    '/CN=kubernetes-admin/O=system:masters' clientAuth
  write_kubeconfig "$pki/admin" kubernetes-admin "$kubernetes/admin.conf"
  issue_certificate "$pki/controller-manager" "$ca" \
    '/CN=system:kube-controller-manager' clientAuth
  write_kubeconfig "$pki/controller-manager" system:kube-controller-manager \
    "$kubernetes/controller-manager.conf"
  issue_certificate "$pki/scheduler" "$ca" \
    '/CN=system:kube-scheduler' clientAuth
  write_kubeconfig "$pki/scheduler" system:kube-scheduler "$kubernetes/scheduler.conf"
  issue_certificate "$pki/kubelet-client" "$ca" \
    "/CN=system:node:$node/O=system:nodes" clientAuth
  write_kubeconfig "$pki/kubelet-client" "system:node:$node" "$kubernetes/kubelet.conf"
}
```

Prepare the first control plane using its reserved address.  The generated
hostname is also installed by Ignition so it matches the certificate identity.

```bash
prepare_control_plane control-plane-1 "$CONTROL_PLANE_1_IP"
```

The resulting `control-plane-1/files/etc/kubernetes` directory contains the
Kubernetes PKI and component kubeconfigs in the layout required on the node.
Each additional control plane needs its own credentials, generated with its
hostname and IP address.

## Embed the Credentials in Ignition

Define a function that packages the files for one node.  Each source file's
path relative to `files` becomes its absolute path on the node.  For example,
`files/etc/kubernetes/pki/ca.crt` becomes `/etc/kubernetes/pki/ca.crt`.

```bash
write_ignition() {
  local node=$1
  local directory="$WORK_DIR/$node/files"
  local file

  echo "Embedding credentials in $node.extra.ign"
  while IFS= read -r -d '' file; do
    echo "Adding /${file#"$directory/"}" >&2
    jq -n --arg path "/${file#"$directory/"}" --rawfile contents "$file" \
      '{path: $path, mode: 384,
        contents: {source: ("data:;base64," + ($contents | @base64))}}'
  done < <(find "$directory" -type f -print0) |
    jq -s '{ignition: {version: "3.4.0"}, storage: {files: .}}' \
      > "$WORK_DIR/$node.extra.ign"

  echo "Writing the cluster configuration for $node"
  NODE_EXTRA="$WORK_DIR/$node.extra.ign" \
    yq '.extraIgnition = strenv(NODE_EXTRA)' "$CLUSTER_CONFIG" \
      > "$WORK_DIR/$node.yaml"
}

write_ignition control-plane-1
```

The generated cluster configuration keeps the settings from `byo.yaml` and
adds `extraIgnition` pointing to the node's credential file.

## Create the Cluster

Generate the complete Ignition configuration for the first control plane.

```bash
ocne cluster start -c control-plane-1.yaml > control-plane-1.ign
create_domain control-plane-1 "$CONTROL_PLANE_1_IP" 52:54:00:cc:01:01
```

The VM uses the `.ign` file emitted by `ocne`, which includes the credentials
from `.extra.ign` as well as the cluster bootstrap configuration.

Follow the first control plane's bootstrap on its console.  Wait for bootstrap
to finish, then press `Ctrl+]` to leave the console.

```bash
sudo virsh console control-plane-1
```

Once the API is available, install the generated administrator kubeconfig on
the administration host.  Do this after generating the initial Ignition and
before running `ocne cluster start` again.

```bash
mkdir -p "$HOME/.kube"
export KUBECONFIG="$HOME/.kube/kubeconfig.$CLUSTER_NAME"
install -m 0600 control-plane-1/files/etc/kubernetes/admin.conf "$KUBECONFIG"
```

## Install Cluster Applications

UI certificates are generated automatically.  To supply them separately,
enable `customUICertificates` in `control-plane-1.yaml` before starting the
applications, then follow [Custom UI Certificates](custom-ui-certificates.md).

Install the cluster applications.

```bash
ocne cluster start -c control-plane-1.yaml --auto-start-ui=false
```

Configure Flannel to use each node's own IP address for tunnels.  The control
plane interface also carries the floating API virtual IP.

```bash
cat > flannel-interface.yaml <<'YAML'
spec:
  template:
    spec:
      containers:
      - name: kube-flannel
        command:
        - /opt/bin/flanneld
        - --ip-masq
        - --kube-subnet-mgr
        - --iface=$(FLANNEL_NODE_IP)
        args: []
        env:
        - name: FLANNEL_NODE_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
YAML
kubectl -n kube-flannel patch daemonset kube-flannel-ds --type=strategic \
  --patch-file flannel-interface.yaml
```

## Join a Control Plane Node

Prepare credentials for the new control plane, generate its join Ignition,
and start its VM.

```bash
prepare_control_plane control-plane-2 "$CONTROL_PLANE_2_IP"
write_ignition control-plane-2
ocne cluster join --role-control-plane -c control-plane-2.yaml > control-plane-2.ign
create_domain control-plane-2 "$CONTROL_PLANE_2_IP" 52:54:00:cc:01:02
```

The CLI creates the join token because `automaticTokenCreation` is enabled.
Ignition supplies the certificates.  Wait for this control plane to finish
joining before starting another node.  Its console is available with
`sudo virsh console control-plane-2`.

## Join a Worker Node

Workers need the public Kubernetes CA certificate and a kubelet client
kubeconfig.  Define a function that uses OpenSSL to issue a client certificate
with `CN=system:node:<hostname>, O=system:nodes`, then embeds it in a kubeconfig.

```bash
prepare_worker() {
  local node=$1
  local directory="$WORK_DIR/$node/files/etc/kubernetes"

  echo "Preparing the CA certificate and kubelet credentials for $node"
  mkdir -p "$directory/pki"
  cp "$WORK_DIR/pki/ca.crt" "$directory/pki/ca.crt"
  printf '%s\n' "$node" > "$WORK_DIR/$node/files/etc/hostname"
  issue_certificate "$directory/pki/kubelet-client" "$WORK_DIR/pki/ca" \
    "/CN=system:node:$node/O=system:nodes" clientAuth
  write_kubeconfig "$directory/pki/kubelet-client" "system:node:$node" \
    "$directory/kubelet.conf"
}

prepare_worker worker-1
write_ignition worker-1
ocne cluster join -c worker-1.yaml > worker-1.ign
create_domain worker-1 "$WORKER_IP" 52:54:00:cc:01:03
```

Worker Ignition contains no CA signing keys.  Its console is available with
`sudo virsh console worker-1`.  To add more nodes, repeat the corresponding
steps with a unique hostname, IP address, and MAC address.
