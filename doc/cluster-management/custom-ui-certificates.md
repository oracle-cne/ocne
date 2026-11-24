# Custom UI Certificates

Set `customUICertificates: true` to supply the UI TLS Secrets separately.
The CLI installs the UI without generating certificates or waiting for it to
start.  Automatic port forwarding and browser startup are also skipped, even
when `autoStartUI` is enabled.  Apply the Secrets after cluster start finishes.

This option defaults to `false`, works with every provider, and is independent
of `customCertificates`, which controls Kubernetes credentials for BYO clusters.

## Configure the Cluster

The examples use an existing cluster configuration named `clusterConfig.yaml`.
The administration host needs OpenSSL, `ocne`, `kubectl`, and the Mike Farah
implementation of `yq`.

For BYO clusters, complete the first control plane bootstrap and supply the
administrator kubeconfig as described in the
[BYO guide](byo.md#creating-a-cluster-and-joining-nodes) before installing cluster
applications with the start command below.

```bash
yq -i '.customUICertificates = true' clusterConfig.yaml
export CLUSTER_NAME=$(yq -r .name clusterConfig.yaml)
ocne cluster start -c clusterConfig.yaml
```

After cluster start finishes, select the administrator kubeconfig and prepare
a private workspace for the UI credentials.

```bash
export KUBECONFIG=$(ocne cluster show -C "$CLUSTER_NAME")
umask 077
mkdir ui-certificates
cd ui-certificates
```

## Generate the Certificates

Create a UI authority and server certificate with OpenSSL.  The example
certificates are valid for one year.  The server certificate covers the UI
Service and access through a localhost port-forward.  Add any other DNS names
used to access the UI to `subjectAltName`.

To use an existing certificate, place the server certificate, private key, and
CA certificate at `ui/server.crt`, `ui/server.key`, and `ui/ca.crt`, then proceed
to installing the Secrets.

```bash
mkdir ui
echo "Creating the UI certificate authority"
openssl req -x509 -newkey rsa:2048 -nodes -days 365 -sha256 \
  -subj "/CN=$CLUSTER_NAME-ui" \
  -addext 'basicConstraints=critical,CA:TRUE' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign' \
  -keyout ui/ca.key -out ui/ca.crt

echo "Creating the UI server certificate request"
openssl req -new -newkey rsa:2048 -nodes -subj '/CN=ui' \
  -keyout ui/server.key -out ui/server.csr
cat > ui/server.ext <<'EOF_EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:ui,DNS:ui.ocne-system,DNS:ui.ocne-system.svc,DNS:localhost,IP:127.0.0.1
EOF_EXT
openssl x509 -req -in ui/server.csr -CA ui/ca.crt -CAkey ui/ca.key \
  -CAcreateserial -days 365 -sha256 -extfile ui/server.ext -out ui/server.crt
```

## Install the Secrets

Create the Secrets in the namespace installed with the UI.

```bash
echo "Installing the UI certificate and CA certificate"
kubectl -n ocne-system create secret tls ui-tls \
  --cert=ui/server.crt --key=ui/server.key
kubectl -n ocne-system create secret generic certificate-authority-tls \
  --from-file=tls.crt=ui/ca.crt
```

The UI starts once the Secrets are available.
