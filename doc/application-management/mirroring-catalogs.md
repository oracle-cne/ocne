# Mirroring Catalogs

When installing applications into clusters without internet access, such as
those in air gapped environments, it is necessary to copy any required container
images into a registry that is accessible to the cluster nodes.  The Oracle
Cloud Native Environment CLI can generate a list of container images that are
required to install applications from a catalog and copy them to another
container registry.  It is possible to clone all the images that can be
deployed from a catalog as well as a subset of images based on a list of
applications and their configuration.

## Listing Images

A list of container images used by a catalog can be generated with

```
$ ocne catalog mirror
```

## Mirroring the Oracle Application Catalog

The Oracle Application Catalog can be mirrored with a simple command.

```
$ ocne catalog mirror --push --destination myregistry.com
```

## Mirroring Specific Applications

If a cluster configuration is provided, only applications listed in the file
are mirrored.

```
$ ocne catalog mirror --push --destination myregistry.com -c clusterconfig.yaml
```

## Mirroring Other Catalogs

Any catalog installed into a cluster can be mirrored using its name.

```
$ ocne catalog mirror --push --destination myregistry.com -n mycatalog
```

## Example

This example starts a private container registry and mirrors the complete Oracle
Application Catalog.  Setting up the private registry is almost the entire
process.  Once the registry is available, actually mirroring the images is a
single command.

### Create the Registry Cluster

Deploy a cluster with MetalLB installed.  Later on, a container registry is
deployed with a LoadBalancer service to expose it to the network outside the
cluster.  MetalLB is used to assign an IP address to the LoadBalancer service.

#### Start the Cluster

```
$ ocne cluster start -c <( echo "
name: registry
headless: true
applications:
  - name: metallb
    release: metallb
    namespace: metallb
"
)
INFO[2024-08-15T18:31:07Z] Creating new Kubernetes cluster named registry 
INFO[2024-08-15T18:32:05Z] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-08-15T18:32:06Z] Installing flannel into kube-flannel: ok 
INFO[2024-08-15T18:32:06Z] Installing app-catalog into ocne-system: ok 
INFO[2024-08-15T18:32:27Z] Waiting for Oracle Catalog to start: ok 
INFO[2024-08-15T18:32:28Z] Installing metallb into metallb: ok 
INFO[2024-08-15T18:32:28Z] Kubernetes cluster was created successfully

$ export KUBECONFIG=$(ocne cluster show -C registry)
$ kubectl wait pod --for=condition=Ready --namespace=metallb --selector='app.kubernetes.io/name=metallb' --timeout=5m
pod/metallb-controller-5984d4dc7b-c9sdv condition met
pod/metallb-speaker-6g2bm condition met
```

#### Create an Address Pool

Create an address pool with a single IP address.  MetalLB will use this pool
to assign an address to the registry service.  The pool contains only a single
address to make it easy to know how to access it in later steps.

The IP address used is only an example.  It is necessary to choose an address
that works in your environment.

```
$ export IP=192.168.122.250

$ kubectl apply -f - << EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: registry-pool
  namespace: metallb
spec:
  addresses:
  - $IP/32
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: registry
  namespace: metallb
spec:
  ipAddressPools:
  - registry-pool
EOF
ipaddresspool.metallb.io/registry-pool created
l2advertisement.metallb.io/registry created
```

### Create the Registry

A registry is deployed into the cluster with a self-signed certificate using a
Helm chart.  The self-signed certificate is added to the trusted certificate
store on the local system so that later steps can verify the authenticity of
the registry.

#### Generate the Self Signed Certificate

Create the certificate and add it to the cluster as well as the local trust store.

```
$ cat > csr.conf << EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = US
ST = CA
L = Redwood City
O = ocne
OU = ocne
CN = registry

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
IP.1 = ${IP}

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:TRUE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF

$ sh << EOF
openssl genrsa -out ca.key 2048
openssl req -new -key ca.key -out ca.csr -config csr.conf
openssl x509 -req -signkey ca.key -in ca.csr -out ca.crt -days 10000 -extensions v3_ext -extfile csr.conf -sha256
EOF
Certificate request self-signature ok
subject=C = US, ST = CA, L = Redwood City, O = ocne, OU = ocne, CN = registry

sudo cp ca.crt /etc/pki/tls/certs/registry.crt
```

#### Install the Registry

Install the self-signed certificate into the cluster and then install the
registry itself.  The giant base64 encoded string is a Helm chart packaged
as a bzip2 compressed tar archive.  It deploys the
container-registry.oracle.com/os/registry:v2.7.1.1 container image.

```
$ kubectl create namespace registry
namespace/registry created

$ kubectl apply -f - << EOF
apiVersion: v1
kind: Secret
metadata:
  name: certs
  namespace: registry
type: Opaque
data:
  ca.crt: $(base64 -w 0 < ca.crt)
  ca.key: $(base64 -w 0 < ca.key)
EOF
secret/certs created

$ base64 -d << EOF | bunzip2 -c | gzip -c > registry.tgz
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
EOF

$ helm install registry registry.tgz --namespace registry
NAME: registry
LAST DEPLOYED: Thu Aug 15 18:32:53 2024
NAMESPACE: registry
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
     NOTE: It may take a few minutes for the LoadBalancer IP to be available.
           You can watch the status of by running 'kubectl get --namespace registry svc -w registry'
  export SERVICE_IP=$(kubectl get svc --namespace registry registry --template "{{ range (index .status.loadBalancer.ingress 0) }}{{.}}{{ end }}")
  echo http://$SERVICE_IP:5000

$ kubectl wait pod --for=condition=Ready --namespace=registry --selector='app.kubernetes.io/name=registry' --timeout=5m
pod/registry-5fd4f condition met
```

### Mirroring the Oracle Catalog

With the registry available, mirror the entire Oracle Application Catalog.

```
$ ocne catalog mirror --push --destination ${IP}:5000
Getting image source signatures
Copying blob afe4d02978bc done   | 
Copying blob 4ae6628a2705 done   | 
Copying blob f1a6f43f4e39 done   | 
Copying blob 39240bd3c42d done   | 
Copying blob 2179b642e6b3 done   | 
Copying blob 17c140712f5f done   | 
Copying blob d19b07a73cbe done   | 
Copying blob 1905671905d7 done   | 
Copying config 680cfb03cb done   | 
Writing manifest to image destination
INFO[2024-08-15T18:33:26Z] Successfully copied image docker://container-registry.oracle.com/olcne/prometheus:v2.31.1 
Getting image list signatures
Copying 6 images generated from 6 images in list
Copying image sha256:e6db7760ee7c55a780001fcf4be69451349c004e3730179c8e61657b0677229e (1/6)
Getting image source signatures
```
