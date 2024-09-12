# Configuration

In Oracle Cloud Native Environment, clusters and applications are configured
through a set of configuration files and command line arguments.  Configuration
is layered, with each layer of configuration taking precendence over the
previous layer.  The layered structure allows for convenient re-use of
parameters that would otherwise have to be duplicated into every deployment.

## Layer Hierarchy

There are four ways to configure `ocne` subcommands.  The methods are
hierarchical, with some layers taking precendence over other.  These are the
methods, listed in hierarchical order:

- Global defaults
- ~/.ocne/defaults.yaml
- Cluster and application configuration files
- `ocne` command line options

Global defaults can be overridden by a global configuration file.  Those values
can in turn be overridden by a cluster or application specific configuration
file.  Finally, that entire stack of configuration can be overridden by command
line parameters.

For detailed documentation of `~/.ocne/defaults.yaml`, refer to
`ocne-defaults.yaml(5)`.  For detailed documentation of cluster and application
specific configuration file, refer to `ocne-config.yaml(5)`.

## Configuring Global Default Values

The contents of `~/.ocne/defaults.yaml` overrides any default parameters.  The
values in this file are used for all `ocne` subcommands unless they are
specfically overriden by a layer with higher precendence.

### Example

In this example, a proxy is configured for all clusters that are started.

In the configuration file, some proxy values are set.
```
$ cat ~/.ocne/default.yaml

```

When a new cluster is created, those values are propagated into the cluster.
```
# Create a new cluster
$ ocne cluster start
INFO[0000] Starting Cluster                             
INFO[0000] Connecting to qemu:///session?socket=/Users/dkrasins/.cache/libvirt/libvirt-sock 
INFO[0000] Creating new Kubernetes cluster named ocne   
INFO[0000] Kubernetes API Server address is 127.0.0.1   
INFO[0000] Tunnel port is 6443                          
INFO[0000] Initializing a cluster with first node ocne-control-plane-1 
INFO[0000] Generating Ignition file                     
INFO[0000] Ensuring presence of storage pool            
INFO[0000] Checking for existance of base image         
INFO[0000] Checking if volume ocne-control-plane-1.qcow2 exists 
INFO[0000] Creating volume ocne-control-plane-1.qcow2   
INFO[0000] Uploading ignition file to ocne-control-plane-1-init.ign 
INFO[0000] Refreshing storage pools                     
INFO[0000] Creating domain ocne-control-plane-1         
INFO[0011] waiting for the Kubernetes cluster to be ready 
...

# Set the kubeconfig
$ export KUBECONFIG=~/.kube/kubeconfig.ocne.local

# Get an admin container
$ export NODE_NAME=$(kubectl get node -o=jsonpath='{.items[0].metadata.name}')
$ kubectl create ns ocne-admin
$ cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: ocne-admin-${NODE_NAME}
  name: ocne-admin-${NODE_NAME}
  namespace: ocne-admin
spec:
  replicas: 1
  selector:
    matchLabels:
      run: ocne-admin-${NODE_NAME}
  template:
    metadata:
      labels:
        run: ocne-admin-${NODE_NAME}
    spec:
      nodeName: ${NODE_NAME}
      hostNetwork: true
      hostPID: true
      tolerations:
      - key: "node-role.kubernetes.io/control-plane"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/unschedulable"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/unreachable"
        operator: "Exists"
        effect: "NoSchedule"
      hostNetwork: true
      hostPID: true
      containers:
      - image: container-registry.oracle.com/os/oraclelinux:8
        name: olwithvolume
        command: ["sleep", "10d"]
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /hostroot
          name: host-root
      volumes:
      - name: host-root
        hostPath:
          path: /
          type: Directory
EOF
$ while ! kubectl -n ocne-admin exec -ti $(kubectl -n ocne-admin get pod -l run=ocne-admin-$NODE_NAME -o jsonpath='{.items[0].metadata.name}') -- sh; do
        sleep 1
done

# See the proxy configuration for cri-o
$ cat /hostroot/etc/systemd/system/crio.service.d/proxy.conf 
[Service]
Environment=HTTPS_PROXY=http://proxy.myorg.com:80
Environment=https_proxy=http://proxy.myorg.com:80
Environment=no_proxy=127.0.0.1,localhost 
```

## Cluster and Application Specific Configuration Files

Most deployments contain at least some details that are unique to that
deployment.  A configuration file can be used to provide those customizations
without having to specify them on the command line.

### Example

A configuration file is used to create a cluster on a remote system

This configuration file sets the target system for the libvirt provider.
```
$ cat remote.yaml
cat remote.yaml 
providers:
  libvirt:
    uri: qemu+ssh://user@host/system
```

When the cluster is started, it will be started on that system.
```
# Start the cluster.  Notice that the connection URI has changed.
dkrasins@dkrasins-mac ocne % ./out/darwin_amd64/ocne cluster start -c remote.yaml
INFO[0000] Starting Cluster                             
INFO[0000] Connecting to qemu+ssh://user@host/system 
INFO[0000] Creating new Kubernetes cluster named ocne   
INFO[0000] Kubernetes API Server address is 192.168.124.201 
INFO[0000] Tunnel port is 6444                          
INFO[0001] Initializing a cluster with first node ocne-control-plane-1 
...

# Check that the proxy configuration from ~/.ocne/defaults.yaml
# is preserved.
$ export NODE_NAME=$(kubectl --insecure-skip-tls-verify get node -o=jsonpath='{.items[0].metadata.name}')
$ kubectl --insecure-skip-tls-verify create ns ocne-admin
$ cat <<EOF | kubectl --insecure-skip-tls-verify apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: ocne-admin-${NODE_NAME}
  name: ocne-admin-${NODE_NAME}
  namespace: ocne-admin
spec:
  replicas: 1
  selector:
    matchLabels:
      run: ocne-admin-${NODE_NAME}
  template:
    metadata:
      labels:
        run: ocne-admin-${NODE_NAME}
    spec:
      nodeName: ${NODE_NAME}
      hostNetwork: true
      hostPID: true
      tolerations:
      - key: "node-role.kubernetes.io/control-plane"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/unschedulable"
        operator: "Exists"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/unreachable"
        operator: "Exists"
        effect: "NoSchedule"
      hostNetwork: true
      hostPID: true
      containers:
      - image: container-registry.oracle.com/os/oraclelinux:8
        name: olwithvolume
        command: ["sleep", "10d"]
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /hostroot
          name: host-root
      volumes:
      - name: host-root
        hostPath:
          path: /
          type: Directory
EOF

$ while ! kubectl --insecure-skip-tls-verify -n ocne-admin exec -ti $(kubectl --insecure-skip-tls-verify -n ocne-admin get pod -l run=ocne-admin-$NODE_NAME -o jsonpath='{.items[0].metadata.name}') -- sh; do
        sleep 1
done
$ cat /hostroot/etc/systemd/system/crio.service.d/proxy.conf 
[Service]
Environment=HTTPS_PROXY=http://proxy.myorg.com:80
Environment=https_proxy=http://proxy.myorg.com:80
Environment=no_proxy=127.0.0.1,localhost
```

## Command Line Options

Command line options take precendence over all others.

### Example

In this example, the previous configuration files are used.  The libvirt URI is
changed to demonstrate that the cluster configuration file is changed due to
the command line option.

```
$ cat ~/.ocne/defaults.yaml 
proxy:
  httpsProxy: http://proxy.myorg.com:80
  noProxy: 127.0.0.1,localhost
$ cat remote.yaml 
providers:
  libvirt:
    uri: qemu+ssh://user@host/system
$ ocne cluster start -c remote.yaml -s qemu:///session
```
