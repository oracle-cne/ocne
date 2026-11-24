# BYO Clusters with Custom Certificates

The BYO provider can start and join Kubernetes nodes using certificates supplied
by the operator. Set `customCertificates: true` to disable certificate generation
by OCNE. Provision the node credentials before OCNE bootstrap and create the UI
TLS Secrets before installing cluster applications.

This example follows the [BYO certificate setup script](../../bats-tests/scenarios/libvirt/byo-certificates-start.sh):
create the authorities, prepare each node's Ignition, boot the first control
plane, supply the administrator kubeconfig, install the UI credentials and
applications, and join the remaining nodes. Libvirt can supply the machines,
but the cluster configuration must use `provider: byo`.

## Prerequisites

Use an OCK boot image matching the configured Kubernetes minor version and a
means to deliver a different Ignition file to each machine. The [BYO installation
example](byo.md#creating-a-cluster-and-joining-nodes) describes boot image and
machine provisioning. The node counts in the configuration do not create BYO
machines automatically.

The administration host needs `ocne`, `kubectl`, OpenSSL, Bash, `jq`, and the
Mike Farah implementation of `yq`. Each control plane needs its own reachable
IPv4 address. Reserve an unused virtual IP on the control-plane network for the
Kubernetes API. The administration host and every node must reach that address.
Adjust the interface, addresses, subnets, registry access, and proxy settings to
match the installation.

The example creates separate Kubernetes, etcd, front-proxy, and UI authorities.
Its certificates are valid for 30 days, matching the test fixture. For an
installation using an existing PKI, supply the corresponding certificates and
keys and select validity periods and renewal procedures appropriate to that PKI.
This provisioning method places signing keys on control planes so kubeadm and
Kubernetes controllers can use them. Workers receive no CA signing keys.

## Configure the Cluster

Run the administration-host commands in the same Bash session. Create a private
working directory for credentials and Ignition files. Ignition will contain
private keys and must be delivered through a protected provisioning channel.

```bash
set -euo pipefail
umask 077
mkdir byo-custom-certificates
cd byo-custom-certificates
export WORK_DIR="$PWD"
export PKI_DIR="$WORK_DIR/pki"
mkdir -p "$PKI_DIR"
echo "Preparing the BYO certificate workspace in $WORK_DIR"

cat > clusterConfig.yaml <<'YAML'
name: byo-custom-certificates
provider: byo
kubernetesVersion: "1.33"
customCertificates: true
controlPlaneNodes: 3
workerNodes: 3
virtualIp: 192.168.122.241
kubeApiServerBindPort: 6443
kubeApiServerBindPortAlt: 6444
serviceSubnet: 10.96.0.0/12
podSubnet: 10.244.0.0/16
bootVolumeContainerImage: container-registry.oracle.com/olcne/ock
osRegistry: ostree-unverified-registry:container-registry.oracle.com/olcne/ock-ostree
headless: false
providers:
  byo:
    networkInterface: enp1s0
    automaticTokenCreation: true
YAML

export CLUSTER_CONFIG="$WORK_DIR/clusterConfig.yaml"
export CLUSTER_NAME=$(yq -r .name "$CLUSTER_CONFIG")
export API_VIP=$(yq -r .virtualIp "$CLUSTER_CONFIG")
```

`customCertificates` is a top-level cluster option. Its default is `false`,
which enables the normal certificate-generation behavior. The option can also
be set in the existing defaults file; an explicit cluster value overrides the
defaults value. For any provider other than BYO, the option must be omitted
from both files. An explicit `false` is also rejected for those providers.

With `true`, the initial `ocne cluster start` generates Ignition without creating
the administrator kubeconfig or bootstrap PKI. OCNE skips kubeadm's certificate,
kubeconfig, and certificate-upload phases; control-plane joins also skip
certificate download. The flag supplies no certificate paths. Use
`extraIgnition` or another provisioning mechanism to install the required files.

## Provision the Authorities and Administrator Credentials

Generate the four authorities and a shared service-account signing key pair.
Run this once for the cluster, rather than once per node.

```bash
for authority in kubernetes etcd front-proxy ui; do
  echo "Creating the $authority certificate authority"
  openssl req -x509 -newkey rsa:2048 -nodes -days 30 -sha256 \
    -subj "/CN=$CLUSTER_NAME-$authority" \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign' \
    -keyout "$PKI_DIR/$authority.key" -out "$PKI_DIR/$authority.crt"
done

echo "Creating the shared service-account signing keys"
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$PKI_DIR/sa.key"
openssl pkey -in "$PKI_DIR/sa.key" -pubout -out "$PKI_DIR/sa.pub"
```

Define helpers to issue client and server certificates and build kubeconfigs.
The kubeconfigs embed the supplied CA and use the shared API endpoint.

```bash
issue_certificate() {
  local name=$1 authority=$2 subject=$3 extensions=$4
  echo "Issuing $name from the $authority authority"
  openssl req -new -newkey rsa:2048 -nodes -subj "$subject" \
    -keyout "$PKI_DIR/$name.key" -out "$PKI_DIR/$name.csr"
  printf '%s\n' "$extensions" > "$PKI_DIR/$name.ext"
  openssl x509 -req -in "$PKI_DIR/$name.csr" \
    -CA "$PKI_DIR/$authority.crt" -CAkey "$PKI_DIR/$authority.key" \
    -CAcreateserial -days 30 -sha256 -extfile "$PKI_DIR/$name.ext" \
    -out "$PKI_DIR/$name.crt"
}

write_kubeconfig() {
  local identity=$1 destination=$2
  echo "Writing the kubeconfig for $identity"
  kubectl config --kubeconfig "$destination" set-cluster kubernetes \
    --server="https://$API_VIP:6443" \
    --certificate-authority="$PKI_DIR/kubernetes.crt" --embed-certs=true
  kubectl config --kubeconfig "$destination" set-credentials "$identity" \
    --client-certificate="$PKI_DIR/$identity.crt" \
    --client-key="$PKI_DIR/$identity.key" --embed-certs=true
  kubectl config --kubeconfig "$destination" set-context default \
    --cluster=kubernetes --user="$identity"
  kubectl config --kubeconfig "$destination" use-context default
  chmod 0600 "$destination"
}

CLIENT_EXTENSIONS=$'basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=clientAuth'
issue_certificate admin kubernetes '/CN=kubernetes-admin/O=system:masters' "$CLIENT_EXTENSIONS"
write_kubeconfig admin "$WORK_DIR/admin.conf"

issue_certificate ui-server ui '/CN=ui' \
  $'basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:ui,DNS:ui.ocne-system,DNS:ui.ocne-system.svc,DNS:localhost,IP:127.0.0.1'
```

The administrator identity has cluster-wide privileges. The UI certificate in
this example covers the Service names and a localhost port-forward. Include
any additional DNS names used to access the UI when issuing its certificate.

## Provision Node Credentials Before Bootstrap

Each control plane receives the same authorities and service-account keys.
Its API server and etcd server/peer certificates must be issued for that node's
address and hostname. Reusing one node's complete leaf certificate set on another
node does not provide the required identities.

| Node files | Purpose |
| --- | --- |
| `/etc/kubernetes/pki/ca.crt` and `ca.key` | Kubernetes authority |
| `/etc/kubernetes/pki/etcd/ca.crt` and `ca.key` | etcd authority |
| `/etc/kubernetes/pki/front-proxy-ca.crt` and `front-proxy-ca.key` | Front-proxy authority |
| `/etc/kubernetes/pki/sa.key` and `sa.pub` | Shared service-account signing keys |
| API server, API client, front-proxy client, and etcd leaf certificates and keys | Node-specific component identities |
| `/etc/kubernetes/admin.conf`, `controller-manager.conf`, `scheduler.conf`, and `kubelet.conf` | Component access to the API |

The following first-boot provisioner uses the supplied authorities to issue the
control-plane leaf certificates and kubeconfigs. These explicit kubeadm commands
belong to the operator's provisioning step and finish before OCNE starts. OCNE's
own init and join commands skip those generation phases.

The API VIP listens on port 6443. In this internal load-balancer configuration,
the API server on each control plane binds to port 6444. Keep these ports and
the network ranges consistent with the cluster configuration.

```bash
cat > kubeadm-template.yaml <<YAML
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: NODE_IP
  bindPort: 6444
nodeRegistration:
  criSocket: unix:///var/run/crio/crio.sock
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: KUBERNETES_VERSION
controlPlaneEndpoint: $API_VIP:6443
networking:
  serviceSubnet: $(yq -r .serviceSubnet "$CLUSTER_CONFIG")
  podSubnet: $(yq -r .podSubnet "$CLUSTER_CONFIG")
YAML

cat > provision-certificates.sh <<'BASH'
#!/bin/bash
set -euo pipefail
source /etc/ocne/custom-certificate.env
umask 077
echo "Provisioning custom credentials for $CUSTOM_NODE ($CUSTOM_ROLE)"
if [ "$CUSTOM_ROLE" = control-plane ]; then
  node_ip=$(ip -4 -o addr show dev "$CUSTOM_INTERFACE" scope global |
    awk '{split($4, address, "/"); print address[1]; exit}')
  test -n "$node_ip"
  kube_version=$(kubeadm version -o short)
  echo "Preparing credentials for $CUSTOM_NODE at $node_ip using $kube_version"
  sed -e "s/NODE_IP/$node_ip/g" -e "s/KUBERNETES_VERSION/$kube_version/g" \
    /etc/ocne/custom-kubeadm-template.yaml > /etc/ocne/custom-kubeadm.yaml
  kubeadm init phase certs all --config /etc/ocne/custom-kubeadm.yaml
  kubeadm init phase kubeconfig all --config /etc/ocne/custom-kubeadm.yaml
fi
touch /etc/ocne/custom-certificates-provisioned
echo "Custom credential provisioning is complete for $CUSTOM_NODE"
BASH
```

Run the provisioner on a fresh node before the VIP is assigned to its interface.
It selects the first global IPv4 address on the configured interface. Where an
interface has several node addresses, supply the intended advertise address
explicitly instead. Kubeadm adds the Kubernetes Service identity and the API
endpoint to the API server certificate; add any additional endpoint SANs to
`apiServer.certSANs` in the template before issuing certificates.

Workers need the public Kubernetes CA and a kubelet client kubeconfig. The
worker client subject must be `CN=system:node:<hostname>, O=system:nodes` and
must match the hostname delivered to that worker.

## Prepare Each Node's Ignition

The following functions package the files using generic extra Ignition. The
provisioning service is required by, and ordered before, `ocne.service`. A failed
provisioning step therefore prevents OCNE bootstrap from proceeding with missing
credentials. The completion marker prevents reissuing credentials on every boot.

```bash
add_ignition_file() {
  local source=$1 destination=$2 mode=${3:-384} encoded
  echo "Adding $destination to $NODE_EXTRA"
  encoded=$(base64 -w0 "$source")
  jq --arg path "$destination" --arg data "data:;base64,$encoded" \
    --argjson mode "$mode" \
    '.storage.files += [{path:$path, mode:$mode, contents:{source:$data}}]' \
    "$NODE_EXTRA" > "$NODE_EXTRA.next"
  mv "$NODE_EXTRA.next" "$NODE_EXTRA"
}

prepare_node() {
  local node=$1 role=$2 source target
  echo "Preparing custom credentials and Ignition for $node ($role)"
  NODE_EXTRA="$WORK_DIR/$node.extra.ign"
  printf '{"ignition":{"version":"3.4.0"},"storage":{"files":[]}}\n' > "$NODE_EXTRA"
  printf '%s\n' "$node" > "$WORK_DIR/$node.hostname"
  add_ignition_file "$WORK_DIR/$node.hostname" /etc/hostname 420
  add_ignition_file "$PKI_DIR/kubernetes.crt" /etc/kubernetes/pki/ca.crt
  if [ "$role" = control-plane ]; then
    while read -r source target; do
      add_ignition_file "$PKI_DIR/$source" "/etc/kubernetes/pki/$target"
    done <<'FILES'
kubernetes.key ca.key
etcd.crt etcd/ca.crt
etcd.key etcd/ca.key
front-proxy.crt front-proxy-ca.crt
front-proxy.key front-proxy-ca.key
sa.key sa.key
sa.pub sa.pub
FILES
    add_ignition_file "$WORK_DIR/kubeadm-template.yaml" /etc/ocne/custom-kubeadm-template.yaml
  else
    issue_certificate "$node" kubernetes "/CN=system:node:$node/O=system:nodes" "$CLIENT_EXTENSIONS"
    write_kubeconfig "$node" "$WORK_DIR/$node.kubelet.conf"
    add_ignition_file "$WORK_DIR/$node.kubelet.conf" /etc/kubernetes/kubelet.conf
  fi
  printf 'CUSTOM_NODE=%q\nCUSTOM_ROLE=%q\nCUSTOM_INTERFACE=%q\n' \
    "$node" "$role" "$(yq -r .providers.byo.networkInterface "$CLUSTER_CONFIG")" \
    > "$WORK_DIR/$node.env"
  add_ignition_file "$WORK_DIR/$node.env" /etc/ocne/custom-certificate.env
  add_ignition_file "$WORK_DIR/provision-certificates.sh" /etc/ocne/provision-custom-certificates.sh 448
  jq '.systemd.units = [
    {name:"custom-certificates.service", enabled:true,
     contents:"[Unit]\nDescription=Provision custom Kubernetes credentials\nWants=network-online.target\nAfter=network-online.target\nBefore=ocne.service\nConditionPathExists=!/etc/ocne/custom-certificates-provisioned\n[Service]\nType=oneshot\nExecStart=/etc/ocne/provision-custom-certificates.sh\nRemainAfterExit=yes\n[Install]\nWantedBy=multi-user.target\n"},
    {name:"ocne.service", dropins:[{name:"custom-certificates.conf",
     contents:"[Unit]\nRequires=custom-certificates.service\nAfter=custom-certificates.service\n"}]}
  ]' "$NODE_EXTRA" > "$NODE_EXTRA.next"
  mv "$NODE_EXTRA.next" "$NODE_EXTRA"
  NODE_EXTRA="$NODE_EXTRA" yq '.extraIgnition = strenv(NODE_EXTRA)' "$CLUSTER_CONFIG" \
    > "$WORK_DIR/$node.yaml"
}
```

File modes in Ignition JSON are decimal: `384` is `0600`, `420` is `0644`, and
`448` is `0700`. Only hostname files use the public-readable mode in this example.
If a separate issuer supplies fully prepared leaf certificates and kubeconfigs,
install those files before `ocne.service` instead of running the kubeadm
provisioning phases above.

## Start the First Control Plane

Generate Ignition for the first node before placing `admin.conf` at OCNE's
expected kubeconfig path. Use a new cluster name whose local registration and
kubeconfig do not already exist.

```bash
echo "Generating Ignition for the first control plane"
test ! -e "$HOME/.kube/kubeconfig.$CLUSTER_NAME"
prepare_node "$CLUSTER_NAME-control-plane" control-plane
ocne cluster start -c "$WORK_DIR/$CLUSTER_NAME-control-plane.yaml" \
  > "$WORK_DIR/$CLUSTER_NAME-control-plane.ign"
```

Boot the first machine using the matching OCK image and this final `.ign` file.
The `.extra.ign` file is an input to OCNE, not the complete boot configuration.
Use the boot-image delivery method prepared in the prerequisites. For libvirt,
follow the VM definition and Ignition attachment in the [BYO example](byo.md#creating-a-cluster-and-joining-nodes),
substituting this node's name, disk, and final Ignition path.

Install the supplied administrator kubeconfig locally, then wait for the API.
The example stops after 15 minutes if bootstrap does not complete.

```bash
echo "Installing the supplied administrator kubeconfig"
mkdir -p "$HOME/.kube"
export KUBECONFIG="$HOME/.kube/kubeconfig.$CLUSTER_NAME"
test ! -e "$KUBECONFIG"
install -m 0600 "$WORK_DIR/admin.conf" "$KUBECONFIG"

echo "Waiting for the Kubernetes API at $API_VIP:6443"
deadline=$((SECONDS + 900))
until kubectl --request-timeout=5s get --raw=/readyz; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "The Kubernetes API did not become ready before the deadline" >&2
    exit 1
  fi
  sleep 5
done
```

Continue only after `/readyz` returns `ok`. On a node that fails bootstrap,
inspect `journalctl -u custom-certificates.service -u ocne.service`. Supplying
`kubelet.conf` alone is not a bootstrap completion signal in custom mode; OCNE
records successful bootstrap in `/etc/ocne/bootstrap-complete`.

## Install UI Credentials and Cluster Applications

Create both Secrets before the second `ocne cluster start`. Custom mode also
skips OCNE's UI certificate generation.

```bash
echo "Installing the supplied UI certificate and trust anchor"
kubectl create namespace ocne-system --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ocne-system create secret tls ui-tls \
  --cert="$PKI_DIR/ui-server.crt" --key="$PKI_DIR/ui-server.key"
kubectl -n ocne-system create secret generic certificate-authority-tls \
  --from-file=tls.crt="$PKI_DIR/ui.crt"

echo "Installing the cluster applications"
ocne cluster start -c "$CLUSTER_CONFIG" --auto-start-ui=false
```

The second start uses the existing API and administrator kubeconfig to install
applications. With workers configured, some application pods remain Pending
until workers join because the control planes are tainted.

### Select the Flannel Node Addresses

In the example network, the API VIP shares an interface with the control-plane
node address. Configure Flannel to use each node's own address for VXLAN rather
than selecting the shared VIP. This is a network setting for this topology.

```bash
echo "Selecting each node's address for Flannel tunnels"
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
  --patch-file "$WORK_DIR/flannel-interface.yaml"
kubectl -n kube-flannel rollout status daemonset/kube-flannel-ds --timeout=900s
```

## Join the Remaining Nodes

Define a bounded registration and readiness wait. Transient API errors can
occur while etcd membership changes as a control plane joins.

```bash
wait_for_node() {
  local node=$1 deadline=$((SECONDS + 900))
  echo "Waiting for $node to register and become Ready"
  until kubectl wait --for=create "node/$node" --timeout=15s &&
    kubectl wait --for=condition=Ready "node/$node" --timeout=15s; do
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "$node did not become Ready before the deadline" >&2
      return 1
    fi
    echo "Registration or readiness is not yet confirmed for $node; retrying"
    sleep 5
  done
}
```

Prepare unique credentials and final Ignition for each additional control plane.
The command below illustrates the second control plane; repeat it for
`$CLUSTER_NAME-control-plane-3`.

```bash
node="$CLUSTER_NAME-control-plane-2"
echo "Generating join Ignition for $node"
prepare_node "$node" control-plane
ocne cluster join --role-control-plane -c "$WORK_DIR/$node.yaml" > "$WORK_DIR/$node.ign"
```

Boot that machine with its final Ignition file, then wait before proceeding to
the next control plane:

```bash
wait_for_node "$node"
```

All control planes use the shared authorities but issue their own leaf credentials.
Custom mode does not upload or download a kubeadm certificate bundle.

For each worker, issue the hostname-specific kubelet identity and generate the
worker join Ignition. Repeat for workers 2 and 3, booting each machine with its
own file.

```bash
node="$CLUSTER_NAME-worker-1"
echo "Generating join Ignition for $node"
prepare_node "$node" worker
ocne cluster join -c "$WORK_DIR/$node.yaml" > "$WORK_DIR/$node.ign"
```

After booting the worker with that final Ignition file, run `wait_for_node "$node"`
before preparing the next worker.

`automaticTokenCreation: true` lets the CLI create join tokens through the live
cluster. Generate each join configuration close to the time the node will boot;
join tokens are temporary. Certificate provisioning and token creation are
separate requirements.

## Verify the Cluster and Certificates

After booting all six machines, verify readiness and application availability.

```bash
echo "Checking all six nodes and the cluster applications"
kubectl wait --for=condition=Ready nodes --all --timeout=900s
kubectl get nodes -o wide
kubectl -n kube-system rollout status deployment/coredns --timeout=900s
kubectl -n ocne-system rollout status deployment/ui --timeout=900s
kubectl get --raw=/readyz
```

Confirm that `kubectl get nodes` lists three control planes and three workers.
Verify the live API server certificate against the supplied Kubernetes CA and
the VIP used by the administrator kubeconfig.

```bash
echo "Verifying the API server certificate and endpoint identity"
openssl s_client -connect "$API_VIP:6443" -CAfile "$PKI_DIR/kubernetes.crt" \
  -verify_ip "$API_VIP" -verify_return_error < /dev/null
```

Check a control plane's installed trust anchor against the original authority.
When inspecting multiple nodes in a shell loop, redirect console input from
`/dev/null` so it cannot consume the list of node names.

```bash
echo "Comparing the installed Kubernetes authority with the supplied certificate"
ocne cluster console --direct --node "$CLUSTER_NAME-control-plane" -- \
  cat /etc/kubernetes/pki/ca.crt < /dev/null > "$WORK_DIR/installed-ca.crt"
cmp "$PKI_DIR/kubernetes.crt" "$WORK_DIR/installed-ca.crt"
```

For the UI, start a port-forward in one terminal and verify the connection in a
second terminal with the same working-directory variables. Successful
verification uses the supplied UI authority and the `localhost` identity in the
issued certificate.

```bash
echo "Forwarding the UI TLS port"
kubectl -n ocne-system port-forward service/ui 8443:443
```

```bash
echo "Verifying the certificate served by the running UI"
openssl s_client -connect 127.0.0.1:8443 -servername localhost \
  -CAfile "$PKI_DIR/ui.crt" -verify_hostname localhost \
  -verify_return_error < /dev/null
```

The [live certificate tests](../../bats-tests/tests/functional/cluster/byo-certificates.bats)
also verify every control-plane component and kubelet client, absence of signing
keys on workers, preservation of provisioned credentials during bootstrap, and
equality between the UI's served certificate and its Secret. The
[upgrade tests](../../bats-tests/tests/upgrade/tests.bats) repeat trust checks after
applicable node updates. A same-minor update must wait for a changed boot ID and
Ready status on each node before advancing; its version string alone cannot
confirm a reboot.
