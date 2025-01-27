# Upgrade OpenSearch Dashboards

### Version: v0.0.1-draft

Upgrade from OpenSearch Dashboards 2.3.0 to 2.15.0.

## Modify the OpenSearch Dashboards objects to be annotated as being managed by Helm
Verrazzano does not deploy OpenSearch Dashboards using a Helm chart.
The installed version of OpenSearch Dashboards needs to be transformed to be manageable by Helm.

```text
kubectl -n verrazzano-system label deployment vmi-system-osd app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate deployment vmi-system-osd meta.helm.sh/release-name=opensearch-dashboards
kubectl -n verrazzano-system annotate deployment vmi-system-osd meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label ingress vmi-system-osd app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate ingress vmi-system-osd meta.helm.sh/release-name=opensearch-dashboards
kubectl -n verrazzano-system annotate ingress vmi-system-osd meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label service vmi-system-osd app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate service vmi-system-osd meta.helm.sh/release-name=opensearch-dashboards
kubectl -n verrazzano-system annotate service vmi-system-osd meta.helm.sh/release-namespace=verrazzano-system
```

## Create environment variables
Set some environment variables to be used in the upgrade steps.

 ```text
 export INGRESS_IP=$(kubectl get ingress -n verrazzano-system vmi-system-osd -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
 ```

## Generate the values override file for the Helm deployment
 ```text
envsubst > values.yaml - <<EOF
fullnameOverride: vmi-system 
deployment:
  osd:
    image:
      repository: olcne/opensearch-dashboards
      tag: 2.15.0
    env:
      disableSecurityDashboardsPlugin: "true"
      opensearchHosts: http://vmi-system-os-ingest:9200/
  extraLabels:
    k8s-app: verrazzano.io
    verrazzano-component: osd
    vmo.v1.verrazzano.io: system
  extraTemplateAnnotations:
    proxy.istio.io/config: '{ ''holdApplicationUntilProxyStarts'': true }'
  serviceAccount: verrazzano-monitoring-operator
  extraTemplateLabels:
    verrazzano-component: osd
service:
  extraLabels:
    k8s-app: verrazzano.io
    verrazzano-component: osd
    vmo.v1.verrazzano.io: system
ingress:
  extraLabels:
    k8s-app: verrazzano.io
    vmo.v1.verrazzano.io: system
  extraAnnotations:
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: osd.vmi.system.default.${INGRESS_IP}.nip.io
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.${INGRESS_IP}.nip.io
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
  spec:
    ingressClassName: verrazzano-nginx
    rules:
      host: osd.vmi.system.default.${INGRESS_IP}.nip.io
      service:
        name: verrazzano-authproxy
        port:
          number: 8775
    tls:
      secretName: system-tls-osd
EOF
```

Install the Helm chart.
 ```text
 ocne app install --name opensearch-dashboards --release opensearch-dashboards --namespace verrazzano-system --catalog embedded --version 2.15.0 --values values.yaml
 ```

