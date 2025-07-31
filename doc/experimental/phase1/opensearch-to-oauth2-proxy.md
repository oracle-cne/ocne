# Migration from Verrazzano Auth Proxy to OAuth2 Proxy for OpenSearch and OpenSearch Dashboard.

### Version: v0.0.1-draft
This document explains how to migrate from OpenSearch and OpenSearch Dashboard to the [OAuth2 Proxy](https://github.com/oauth2-proxy/oauth2-proxy).
This migration is a special case, and should only be done if you migrated OpenSearch and OpenSearch Dashboard after
you migrated to OAuth2 proxy as described here: [instructions](../phase1/oauth2-proxy.md).

The assumption is that OAuth2 Proxy has already been installed on the system.  If you have not installed OAuth2 Proxy
then follow these [instructions](../phase1/oauth2-proxy.md) instead of using this document.

## Summary of steps
1. Shutdown Verrazzano auth proxy.
2. Delete Existing Ingresses.
3. Migrate OpenSearch to use OAuth2 Proxy.
4. Migrate OpenSearch Dashboard to use OAuth2 Proxy.
5. Remove Verrazzano auth proxy from the cluster.

## Prerequisites
### Define the $INGRESS_HOST environment variable
The section uses the INGRESS_HOST environment variable so you must define it.  For example:
The INGRESS_HOST for `https://opensearch.vmi.system.default.11.22.33.44.nip.io` is `11.22.33.44.nip.io`.
So you would run the following in this case:
```text
INGRESS_HOST=11.22.33.44.nip.io
```
**WARNING: Defining INGRESS_HOST is a critical step, which is required by this migration document.  Be sure to do this correctly.**

## 1. Shutdown Verrazzano auth-proxy
Shutdown the Verrazzano auth-proxy by scaling the replicas to 0 as follows:
```
kubectl scale deployment -n verrazzano-system verrazzano-authproxy --replicas 0
```
Verify the pods have been stopped:
```
kubectl get deployment -n verrazzano-system  verrazzano-authproxy
```
Output:
```
NAME                   READY   ...
verrazzano-authproxy   0/0     ...
```

## 2. Delete existing ingresses
### Save existing ingress manifests
```text
kubectl get ingress -n verrazzano-system vmi-system-os-ingest -o yaml > save-ingress-os-ingest.yaml
kubectl get ingress -n verrazzano-system vmi-system-osd -o yaml > save-ingress-osd.yaml
```

### Delete ingresses
```text
kubectl delete ingress -n verrazzano-system vmi-system-os-ingest
kubectl delete ingress -n verrazzano-system vmi-system-osd
```

**WARNING** After migrating each component, you MUST test the component console using a browser to ensure that it is working.
Use the command `vz status` to see the console URLS.


## 3. Migrate OpenSearch to use OAuth2 Proxy
**NOTE**The entire YAML needs to be applies since strategic patches do not work correctly for adding entries to arrays for certain resources.

Update the NetworkPolicy:
```text
cat <<'EOF' > ./opensearch-netpol.yaml 
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
cat <<'EOF' > ./opensearch-authpol.yaml 
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

Create the Ingresses:
```text
cat <<'EOF' > ./opensearch-ingress.yaml 
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-os-ingest-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: opensearch.vmi.system.default.INGRESS_HOST
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
    - opensearch.vmi.system.default.INGRESS_HOST
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
    cert-manager.io/common-name: opensearch.vmi.system.default.INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.INGRESS_HOST
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
  - host: opensearch.vmi.system.default.INGRESS_HOST
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

Update the YAML file and apply it:
```
sed -i -e "s/INGRESS_HOST/$INGRESS_HOST/g"  ./opensearch-ingress.yaml
kubectl apply -f ./opensearch-ingress.yaml
```

### 4. Migrate OpenSearch Dashboard to OAuth2 Proxy
Update the NetworkPolicy:
```text
cat <<'EOF' > ./osd-netpol.yaml 
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
cat <<'EOF' > ./osd-authpol.yaml 
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
cat <<'EOF' > ./osd-ingress.yaml 
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vmi-system-osd-external
  namespace: verrazzano-system
spec:
  ingressClassName: verrazzano-nginx
  rules:
  - host: osd.vmi.system.default.INGRESS_HOST
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
    - osd.vmi.system.default.INGRESS_HOST
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
    cert-manager.io/common-name: osd.vmi.system.default.INGRESS_HOST
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.INGRESS_HOST
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
  - host: osd.vmi.system.default.INGRESS_HOST
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

Update the YAML file and apply it:
```
sed -i -e "s/INGRESS_HOST/$INGRESS_HOST/g"  ./osd-ingress.yaml
kubectl apply -f ./osd-ingress.yaml
```

## 5. Remove Verrazzano Auth Proxy from the system
```text
helm delete -n verrazzano-system verrazzano-authproxy 
```
Output:
```text
release "verrazzano-authproxy" uninstalled
```

## Summary
At this point, traffic from NGINX Ingress Controller now uses the OAuth2 Proxy for OIDC instead of the
Verrazzano Auth Proxy for OpenSearch and OpenSearch Dashboard. The Verrazzano proxy has been removed from the system.  


