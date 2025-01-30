# Migration from Verrazzano Auth proxy to OAuth2 Proxy

### Version: v0.0.1-draft
This document explains how to migrate from the Verrazzano Auth proxy to the [OAuth2 Proxy](https://github.com/oauth2-proxy/oauth2-proxy).

## Summary of steps
1. Prepare for installation of OAuth2 Proxy.
2. Install OAuth2 Proxy.
3. Shutdown Verrazzano auth proxy and Verrazzano monitoring operator.
4. NGINX ingress controller configuration changes.
5. Migrate each console to use OAuth2 Proxy.
6. Update Fluentd config.
7. Remove Verrazzano auth proxy from the cluster.

## 1. Prepare for installation of OAuth2 Proxy 
Before installing the OAuth2 Proxy, you must do the following:

1. Create a Keycloak TLS secret.
2. Create a OAuth2 Proxy secret with OIDC credentials.
3. Create a configuration file to be used for installation.
4. Add an email address to the Keycloak client.

### Create the Keycloak TLS secret
The existing Keycloak TLS secret needs to be copied from the keycloak namespace to verrazzano-system.  This is a simple procedure as described below.

First create a file containing the existing secret:
```
kubectl get secret keycloak-tls -n keycloak -o yaml > ./keycloak-oauth2-tls
```

Next edit ./keycloak-oauth2-tls and do the following:
```text
sed -i '/resourceVersion/,+d' ./keycloak-oauth2-tls
sed -i '/uid/,+d' ./keycloak-oauth2-tls
sed -i '/creationTimestamp/,+d' ./keycloak-oauth2-tls
sed -i 's/namespace: keycloak/namespace: verrazzano-system/' ./keycloak-oauth2-tls
```

Create the new secret:
**NOTE** You may see a warning: `...is missing the kubectl.kubernetes.io/last-applied-configuration annotation`.  Ignore this.
```
kubectl apply -f ./keycloak-oauth2-tls
```

Ensure that the secret has been created:
```
kubectl get secret -n verrazzano-system | grep keycloak
```
output:
```
keycloak-oauth2-tls                                     kubernetes.io/tls    3      5s
```

### Create the OAuth2 Proxy credentials secret
The oauth2-proxy pod refers to a secret which contains the following fields:

1. client-id
2. client-secret
3. cookie-secret

#### create the client id
The client id, already exists in Keycloak so this will be used.  You can get the clear text client id as follows:
```
helm get values -n verrazzano-system verrazzano-authproxy | grep OIDCClientID
```
output:
```
OIDCClientID: <client-id>
```

Next, base64 encode the client id.  Replace the <...> section with real value from the previous command:  
**WARNING: Be sure to replace <client-id> with the actual client ID.**
```
CLIENT_ID=$(echo -n <client-id> | base64)
```

#### create the client secret
The client secret is required by the OAuth2 Proxy even though it is not used in this case.  Generate a fake secret using any string, for example:
```
CLIENT_SECRET=$(cat /proc/sys/kernel/random/uuid | base64)
```

#### create the cookie secret
The cookie secret is a binary 32 byte value that must be base64-URL encoded, then that output needs to be base64 encoded again.

```
COOKIE_SECRET=$(openssl rand  32  | base64 | tr '/+' '_-' | tr -d '=' | base64)
```

### create and apply the secret YAML file
Create a secret YAML file, name oauth2-proxy.yaml, with the values from the first 3 steps (replace the <...> sections with real values).

```
envsubst > ./oauth2-secret.yaml - <<EOF
apiVersion: v1
data:
  cookie-secret: $COOKIE_SECRET
  client-id: $CLIENT_ID
  client-secret: $COOKIE_SECRET
kind: Secret
metadata:
  name: oauth2-proxy
  namespace: verrazzano-system
type: Opaque
```

Apply the secret file to create the secret:
```
kubectl apply -f oauth2-secret.yaml
```

Ensure that the secret has been created:
```
kubectl get secret -n verrazzano-system oauth2-proxy
```
output:
```
NAME           TYPE     DATA   AGE
oauth2-proxy   Opaque   3      3d20h
```

### Create OAuth2 Proxy overrides file
Create an overrides file to be used when installing oauth2-proxy from the catalog.
Define the keycloak URL in an environment variable, replacing the <...> section with the real URL.:
```
KEYCLOAK_URL=<keycloak_url>
```

Execute the following command to generate the oauth2-proxy overrides file:
```text
cat <<EOF > ./oauth2-values.yaml 
extraVolumes:
  - name: keycloak-ca-bundle-cert
    secret:
      secretName: keycloak-oauth2-tls

extraVolumeMounts:
  - mountPath: /etc/ssl/certs/keycloak/
    name: keycloak-ca-bundle-cert

customLabels:
  sidecar.istio.io/inject: "false"
service:  
  portNumber: 49000
config:
  existingSecret: oauth2-proxy
  configFile: |-
    provider_ca_files = [ "/etc/ssl/certs/keycloak/tls.crt", "/etc/ssl/certs/keycloak/ca.crt" ]
    email_domains = [ "*" ]
    insecure_oidc_allow_unverified_email = true
    upstreams="file:///dev/null"
    provider = "oidc"
    code_challenge_method = "S256"
    oidc_issuer_url = "$KEYCLOAK_URL/auth/realms/verrazzano-system"
    skip_provider_button = true
    approval_prompt = "auto"
    reverse_proxy = true
    set_xauthrequest = true
    set_authorization_header = true
    pass_user_headers = true
    pass_access_token = true
EOF    
```


### Add email to the Keycloak client
Log into the Keycloak admin console and add an email to the client using the following steps.

1. Get the Keycloak URL:
```
vz status
```
output:
```
Verrazzano Status
...
  Access Endpoints:
...
    keyCloakUrl: <keycloak-url>
```

2. Get the `keycloakadmin` password:
```
kubectl get secret \
    --namespace keycloak keycloak-http \
    -o jsonpath={.data.password} | base64 \
    --decode; echo
```

3. Navigate to the keycloak URL in a browser, and log into Keycloak as user `keycloakadmin`. Select **Clients** 
in the left navigation pane and the list of clients will be shown in the middle pane
under the heading **Client ID**.  Select the correct client then update the email with a valid email and click the Save button.

## 2. Install outh2-proxy from the Catalog
Now you are ready to install the oauth2-proxy from the catalog.  Run the following command:
```
ocne app install -c embedded -n verrazzano-system -N oauth2-proxy -f ./oauth2-values.yaml 
```

Wait for the oauth2 pods to be ready:
```
kubectl rollout status -n verrazzano-system deployment oauth2-proxy -w
```

## 3. Shutdown Verrazzano auth-proxy and Monitoring operator
Shutdown the Verrazzano auth-proxy by scaling the replicas to 0 as follows:
```
kubectl scale deployment -n verrazzano-system verrazzano-authproxy --replicas 0
```
Verify the pods have been stopped:
```
kubectl get deployment -n verrazzano-system  verrazzano-authproxy
```
output:
```
NAME                   READY   ...
verrazzano-authproxy   0/0     ...
```


Shutdown the Verrazzano auth-proxy and monitoring operator by scaling the replicas to 0 as follows:
**NOTE** If the monitoring operator has already been removed from the system then skip this step.
```
kubectl scale deployment -n verrazzano-system verrazzano-monitoring-operator --replicas 0
```
Verify the pods have been stopped:
```
kubectl get deployment -n verrazzano-system  verrazzano-monitoring-operator 
```
output:
```
NAME                   READY   ...
verrazzano-monitoring-operator    0/0     ...
```

## 4. NGINX Ingress Controller Configuration Changes
There are two changes needed for NGINX. First, allow NGINX to communicate with oauth2-proxy which is outside the Istio mesh.
Second, allow the NGINX Ingress controller to process snippets from the Ingress resource.

Patch the NGINX Ingress Controller deployment as follows:
```
kubectl patch configmap -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-controller --type='merge'  --patch '{"data": {"allow-snippet-annotations": "true"}}'
```
output:
```
configmap/ingress-controller-ingress-nginx-controller patched
```

```text
kubectl patch deployment -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-controller --type='merge'  --patch '{"spec": {"template": {"metadata": {"labels": {"traffic.sidecar.istio.io/excludeOutboundPort": "49000"}}}}}'
```
output:
```
deployment.apps/ingress-controller-ingress-nginx-controller patched
```

Ensure that the pod restarted, check the last column, it should be a few minutes (2m is shown below).
```
kubectl get pod -A | grep ingress-controller-ingress-nginx-controller
```
output:
```
verrazzano-ingress-nginx     ingress-controller-ingress-nginx-controller-7f97dd685-gf5gv       2/2     Running   0               2m
```

## 6. Migrate each console to use OAuth2 Proxy

**NOTE**The entire YAML needs to be applies since strategic patches do not work correctly for adding entries to arrays for certain resource.

### Define the $INGRESS_HOST environment variable
The section uses the INGRESS_HOST environment variable so you must define it.  For example:
The INGRESS_HOST for `https://opensearch.vmi.system.default.11.22.33.44.nip.io` is `default.11.22.33.44.nip.io`.
So you would run the following in this case:
```text
INGRESS_HOST=default.11.22.33.44.nip.io
```

### Save existing ingress manifests
kubectl get ingress -n verrazzano-system vmi-system-prometheus -o yaml > save-ingress-prometheus.yaml
kubectl get ingress -n verrazzano-system vmi-system-grafana -o yaml > save-ingress-grafana.yaml
kubectl get ingress -n verrazzano-system vmi-system-os-ingest -o yaml > save-ingress-os-ingest.yaml
kubectl get ingress -n verrazzano-system vmi-system-osd -o yaml > save-ingress-osd.yaml
kubectl get ingress -n verrazzano-system vmi-system-kiali -o yaml > save-ingress-kiali.yaml

### Delete ingresses
kubectl delete ingress -n verrazzano-system vmi-system-prometheus 
kubectl delete ingress -n verrazzano-system vmi-system-grafana 
kubectl delete ingress -n verrazzano-system vmi-system-os-ingest
kubectl delete ingress -n verrazzano-system vmi-system-osd
kubectl delete ingress -n verrazzano-system vmi-system-kiali

**WARNING** After migrating each component, you MUST test the component console using a browser to ensure that it is working.
Use the command `vz status` to see the console URLS.

### Migrate OpenSearch
Update the NetworkPolicy:
```text
cat <<EOF > ./opensearch-netpol.yaml 
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano-network-policies
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-os-ingest
  namespace: verrazzano-system
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-ingress-nginx
    ports:
    - port: 9200
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchLabels:
          app: fluentd
    ports:
    - port: 9200
      protocol: TCP      
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchLabels:
          app: verrazzano-authproxy
    ports:
    - port: 9200
      protocol: TCP
  - from:
    - podSelector:
        matchLabels:
          opensearch.verrazzano.io/role-master: "true"
    - podSelector:
        matchLabels:
          opensearch.verrazzano.io/role-data: "true"
    ports:
    - port: 9300
      protocol: TCP
  - from:
    - podSelector:
        matchLabels:
          app: system-osd
    ports:
    - port: 9200
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - port: 9200
      protocol: TCP
    - port: 15090
      protocol: TCP
  podSelector:
    matchLabels:
      opensearch.verrazzano.io/role-ingest: "true"
  policyTypes:
  - Ingress
EOF
```
Apply the YAML file:
```
kubectl apply -f ./opensearch-netpol.yaml
```

Update the Authorization Policy:
```text
cat <<EOF > ./opensearch-authpol.yaml 
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-es-ingest-authzpol
  namespace: verrazzano-system
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/fluentd
    to:
    - operation:
        ports:
        - "9200"          
  - from:
    - source:
        namespaces:
        - verrazzano-ingress-nginx
        principals:
        - cluster.local/ns/verrazzano-ingress-nginx/sa/ingress-controller-ingress-nginx
    to:
    - operation:
        ports:
        - "9200"              
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy
    to:
    - operation:
        ports:
        - "9200"
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator
    to:
    - operation:
        ports:
        - "9200"
        - "9300"
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus
    to:
    - operation:
        ports:
        - "9200"
        - "15090"
  selector:
    matchLabels:
      app: system-es-ingest
EOF
```
Apply the YAML file:
```
kubectl apply -f ./opensearch-authpol.yaml
```

Create the new Ingress and update the existing one:
```text
envsubst > ./opensearch-ingress.yaml - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-os-ingest-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: opensearch.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /oauth2
        pathType: Prefix
        backend:
          service:
            name: oauth2-proxy
            port:
              number: 49000
  tls:
  - hosts:
    - opensearch.vmi.system.default.$INGRESS_HOST
    secretName: system-tls-os-ingest

---

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: opensearch.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local 
  name: vmi-system-os-ingest
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: opensearch.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /()(.*)
        pathType: ImplementationSpecific
        backend:
          service:
            name: vmi-system-os-ingest 
            port:
              number: 9200
EOF
```

Apply the YAML file:
```
kubectl apply -f ./opensearch-ingress.yaml
```

### Migrate OpenSearch Dashboard
Update the NetworkPolicy:
```text
cat <<EOF > ./osd-netpol.yaml 
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano-network-policies
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-osd
  namespace: verrazzano-system
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-ingress-nginx
    ports:
    - port: 5601
      protocol: TCP
  - from:
    - podSelector:
        matchLabels:
          k8s-app: verrazzano-monitoring-operator
    ports:
    - port: 5601
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - verrazzano-authproxy
          - system-osd
    ports:
    - port: 5601
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - port: 15090
      protocol: TCP
  podSelector:
    matchLabels:
      app: system-osd
  policyTypes:
  - Ingress
EOF
```
Apply the YAML file:
```
kubectl apply -f ./osd-netpol.yaml
```

Update the Authorization Policy:
```text
cat <<EOF > ./osd-authpol.yaml 
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-osd-authzpol
  namespace: verrazzano-system
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        namespaces:
        - verrazzano-ingress-nginx
        principals:
        - cluster.local/ns/verrazzano-ingress-nginx/sa/ingress-controller-ingress-nginx
    to:
    - operation:
        ports:
        - "5601"
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy
    to:
    - operation:
        ports:
        - "5601"
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator
    to:
    - operation:
        ports:
        - "15090"
        - "5601"
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus
    to:
    - operation:
        ports:
        - "15090"
        - "5601"
  selector:
    matchLabels:
      app: system-osd
EOF
```
Apply the YAML file:
```
kubectl apply -f ./osd-authpol.yaml
```

Create the Ingresses:
```text
envsubst > ./osd-ingress.yaml - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-osd-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: osd.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /oauth2
        pathType: Prefix
        backend:
          service:
            name: oauth2-proxy
            port:
              number: 49000
  tls:
  - hosts:
    - osd.vmi.system.default.$INGRESS_HOST
    secretName: system-tls-osd

---

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: osd.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local 
  name: vmi-system-osd
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: osd.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /()(.*)
        pathType: ImplementationSpecific
        backend:
          service:
            name: vmi-system-osd 
            port:
              number: 5601
EOF
```

Apply the YAML file:
```
kubectl apply -f ./osd-ingress.yaml
```

### Migrate Prometheus
Update the NetworkPolicy:
```text
cat <<EOF > ./prometheus-netpol.yaml 
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: prometheus-operator
    meta.helm.sh/release-namespace: verrazzano-monitoring
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-prometheus
  namespace: verrazzano-monitoring
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-ingress-nginx
    ports:
    - port: 9090
      protocol: TCP          
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - verrazzano-authproxy
          - system-grafana
          - kiali
    ports:
    - port: 9090
      protocol: TCP
    - port: 10901
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - jaeger
    ports:
    - port: 9090
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/component: query
    ports:
    - port: 10901
      protocol: TCP
  podSelector:
    matchLabels:
      app.kubernetes.io/name: prometheus
  policyTypes:
  - Ingress 
EOF
```
Apply the YAML file:
```
kubectl apply -f ./prometheus-netpol.yaml
```

Update the Authorization Policy:
```text
cat <<EOF > ./prometheus-authpol.yaml 
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: vmi-system-prometheus-authzpol
  namespace: verrazzano-monitoring
spec:
  rules:
  - from:
    - source:
        namespaces:
        - verrazzano-ingress-nginx
        principals:
        - cluster.local/ns/verrazzano-ingress-nginx/sa/ingress-controller-ingress-nginx
    to:
    - operation:
        ports:
        - "9090"          
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy
        - cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator
        - cluster.local/ns/verrazzano-system/sa/vmi-system-kiali
    to:
    - operation:
        ports:
        - "9090"
        - "10901"
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus
    to:
    - operation:
        ports:
        - "9090"
  selector:
    matchLabels:
      app.kubernetes.io/name: prometheus
EOF
```
Apply the YAML file:
```
kubectl apply -f ./prometheus-authpol.yaml
```

Create the Ingresses:
```text
envsubst > ./prometheus-ingress.yaml - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-prometheus-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: prometheus.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /oauth2
        pathType: Prefix
        backend:
          service:
            name: oauth2-proxy
            port:
              number: 49000
  tls:
  - hosts:
    - prometheus.vmi.system.default.$INGRESS_HOST
    secretName: system-tls-prometheus


---


apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: prometheus.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local 
  name: vmi-system-prometheus
  namespace: verrazzano-monitoring
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: prometheus.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /()(.*)
        pathType: ImplementationSpecific
        backend:
          service:
            name: prometheus-operator-kube-p-prometheus 
            port:
              number: 9090
EOF
```
Apply the YAML file:
```
kubectl apply -f ./prometheus-ingress.yaml
```

### Migrate Grafana
Update the NetworkPolicy:
```text
cat <<EOF > ./grafana-netpol.yaml 
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano-network-policies
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-grafana
  namespace: verrazzano-system
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-ingress-nginx
    ports:
    - port: 3000
      protocol: TCP          
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchLabels:
          app: verrazzano-authproxy
    ports:
    - port: 3000
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - port: 15090
      protocol: TCP
  podSelector:
    matchLabels:
      app: system-grafana
  policyTypes:
  - Ingress
EOF
```
Apply the YAML file:
```
kubectl apply -f ./grafana-netpol.yaml
```

Update the Authorization Policy:
```text
cat <<EOF > ./grafana-authpol.yaml 
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: vmi-system-grafana-authzpol
  namespace: verrazzano-system
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        namespaces:
        - verrazzano-ingress-nginx
        principals:
        - cluster.local/ns/verrazzano-ingress-nginx/sa/ingress-controller-ingress-nginx
    to:
    - operation:
        ports:
        - "3000"              
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy
    to:
    - operation:
        ports:
        - "3000"
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus
    to:
    - operation:
        ports:
        - "15090"
  selector:
    matchLabels:
      app: system-grafana
EOF
```
Apply the YAML file:
```
kubectl apply -f ./grafana-authpol.yaml
```

Create the Ingresses:
```text
envsubst > ./grafana-ingress.yaml - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-grafana-external
  namespace: verrazzano-system
  annotations:
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: grafana.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/service-upstream: "true"
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: grafana.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /oauth2
        pathType: Prefix
        backend:
          service:
            name: oauth2-proxy
            port:
              number: 49000
  tls:
  - hosts:
    - grafana.vmi.system.default.$INGRESS_HOST
    secretName: system-tls-grafana

---

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
    nginx.ingress.kubernetes.io/auth-response-headers: "X-Auth-Request-User, X-Auth_Request-Email"
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: grafana.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local 
    nginx.ingress.kubernetes.io/configuration-snippet: |
      auth_request_set $user   $upstream_http_x_auth_request_preferred_username;
      auth_request_set $email  $upstream_http_x_auth_request_email;
      auth_request_set $token  $upstream_http_x_auth_request_access_token;
      proxy_set_header X-WEBAUTH-USER  $user;
      proxy_set_header X-Email $email;
      proxy_set_header X-Auth-Request-Access-Token $token;
  name: vmi-system-grafana
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: grafana.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /()(.*)
        pathType: ImplementationSpecific
        backend:
          service:
            name: vmi-system-grafana
            port:
              number: 3000
EOF
```

Apply the YAML file:
```
kubectl apply -f ./grafana-ingress.yaml
```

### Migrate Kiali
Update the NetworkPolicy:
```text
cat <<EOF > ./kiali-netpol.yaml 
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  annotations:
    meta.helm.sh/release-name: verrazzano-network-policies
    meta.helm.sh/release-namespace: verrazzano-system
  labels:
    app.kubernetes.io/managed-by: Helm
  name: kiali
  namespace: verrazzano-system
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-ingress-nginx
    ports:
    - port: 20001
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-system
      podSelector:
        matchLabels:
          app: verrazzano-authproxy
    ports:
    - port: 20001
      protocol: TCP
  - from:
    - namespaceSelector:
        matchLabels:
          verrazzano.io/namespace: verrazzano-monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - port: 9090
      protocol: TCP
    - port: 15090
      protocol: TCP
  podSelector:
    matchLabels:
      app: kiali
  policyTypes:
  - Ingress
EOF
```
Apply the YAML file:
```
kubectl apply -f ./kiali-netpol.yaml
```

Update the Authorization Policy:
```text
cat <<EOF > ./kiali-authpol.yaml 
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: vmi-system-kiali-authzpol
  namespace: verrazzano-system
spec:
  rules:
  - from:
    - source:
        namespaces:
        - verrazzano-ingress-nginx
        principals:
        - cluster.local/ns/verrazzano-ingress-nginx/sa/ingress-controller-ingress-nginx
    to:
    - operation:
        ports:
        - "20001"           
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy
    to:
    - operation:
        ports:
        - "20001"
  - from:
    - source:
        namespaces:
        - verrazzano-system
        principals:
        - cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator
    to:
    - operation:
        ports:
        - "9090"
  selector:
    matchLabels:
      app: kiali
EOF
```
Apply the YAML file:
```
kubectl apply -f ./kiali-authpol.yaml
```

Create the Ingresses:
```text
envsubst > ./kiali-ingress.yaml - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-kiali-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: kiali.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /oauth2
        pathType: Prefix
        backend:
          service:
            name: oauth2-proxy
            port:
              number: 49000
  tls:
  - hosts:
    - kiali.vmi.system.default.$INGRESS_HOST
    secretName: system-tls-kiali

---

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: kiali.vmi.system.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_HOST
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 6M
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    nginx.ingress.kubernetes.io/service-upstream: "true"
    nginx.ingress.kubernetes.io/upstream-vhost: ${service_name}.${namespace}.svc.cluster.local 
  name: vmi-system-kiali
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: kiali.vmi.system.default.$INGRESS_HOST
    http:
      paths:
      - path: /()(.*)
        pathType: ImplementationSpecific
        backend:
          service:
            name: vmi-system-kiali
            port:
              number: 20001
EOF
```

Apply the YAML file:
```
kubectl apply -f ./kiali-ingress.yaml
```

### Migrate Fluentd
Get the Fluentd DaemonSet YAML
```text
kubectl get daemonset fluentd -n verrazzano-system -o yaml >  ./fluentd-oauth2-daemonset.yaml
cp ./fluentd-oauth2-daemonset.yaml ./fluentd-daemonset-save.yaml
```
Update YAML:
```text
sed -i '/resourceVersion/,+d' ./fluentd-oauth2-daemonset.yaml
sed -i '/uid/,+d' ./fluentd-oauth2-daemonset.yaml
sed -i '/ELASTICSEARCH_USER/,+5d' ./fluentd-oauth2-daemonset.yaml
sed -i '/ELASTICSEARCH_PASSWORD/,+5d' ./fluentd-oauth2-daemonset.yaml
sed -i 's/verrazzano-authproxy-opensearch:8775/vmi-system-os-ingest:9200/' ./fluentd-oauth2-daemonset.yaml
```

Apply the YAML:
```text
kubectl apply -f fluentd-daemonset.yaml
cp fluentd-daemonset.yaml fluentd-daemonset-save.yaml
```

Wait for the Fluentd pods to be ready:
```
kubectl rollout status -n verrazzano-system daemonset fluentd -w
```
Output:
```text
daemon set "fluentd" successfully rolled out
```

Delete the Fluentd OpenSearch configmap
```text
kubectl delete configmap -n verrazzano-system  fluentd-es-config 
```
Output:
```text
configmap "fluentd-es-config" deleted
```

Delete the Fluentd OpenSearch secret
```text
kubectl delete secret -n verrazzano-system verrazzano-es-internal
```
Output:
```text
secret "verrazzano-es-internal" deleted
```

