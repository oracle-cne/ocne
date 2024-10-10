# Upgrade Istio

### Version: v0.0.1-draft

Upgrade from Istio 1.19.3 to 1.19.9.

## Modify the Istio objects to be annotated as being managed by Helm

Verrazzano does not deploy Istio using a Helm chart.
The installed version of Istio needs to be transformed to be manageable by Helm.

```text
# istio-base
kubectl -n istio-system label ServiceAccount istio-reader-service-account app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ServiceAccount istio-reader-service-account meta.helm.sh/release-name=istio-base
kubectl -n istio-system annotate ServiceAccount istio-reader-service-account meta.helm.sh/release-namespace=istio-system
 
kubectl label ValidatingWebhookConfiguration istiod-default-validator app.kubernetes.io/managed-by=Helm
kubectl annotate ValidatingWebhookConfiguration istiod-default-validator meta.helm.sh/release-name=istio-base
kubectl annotate ValidatingWebhookConfiguration istiod-default-validator meta.helm.sh/release-namespace=istio-system

# istiod
kubectl -n istio-system label ServiceAccount istiod app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ServiceAccount istiod meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate ServiceAccount istiod meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label ConfigMap istio app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ConfigMap istio meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate ConfigMap istio meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label ConfigMap istio-sidecar-injector app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ConfigMap istio-sidecar-injector meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate ConfigMap istio-sidecar-injector meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRole istiod-clusterrole-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRole istiod-clusterrole-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRole istiod-clusterrole-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRole istiod-gateway-controller-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRole istiod-gateway-controller-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRole istiod-gateway-controller-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRole istio-reader-clusterrole-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRole istio-reader-clusterrole-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRole istio-reader-clusterrole-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRoleBinding istiod-clusterrole-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRoleBinding istiod-clusterrole-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRoleBinding istiod-clusterrole-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRoleBinding istiod-gateway-controller-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRoleBinding istiod-gateway-controller-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRoleBinding istiod-gateway-controller-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl label ClusterRoleBinding istio-reader-clusterrole-istio-system app.kubernetes.io/managed-by=Helm
kubectl annotate ClusterRoleBinding istio-reader-clusterrole-istio-system meta.helm.sh/release-name=istiod
kubectl annotate ClusterRoleBinding istio-reader-clusterrole-istio-system meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label Role istiod app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Role istiod meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate Role istiod meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label RoleBinding istiod app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate RoleBinding istiod meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate RoleBinding istiod meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label Service istiod app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Service istiod meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate Service istiod meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label Deployment istiod app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Deployment istiod meta.helm.sh/release-name=istiod
kubectl -n istio-system annotate Deployment istiod meta.helm.sh/release-namespace=istio-system

kubectl label MutatingWebhookConfiguration istio-sidecar-injector app.kubernetes.io/managed-by=Helm
kubectl annotate MutatingWebhookConfiguration istio-sidecar-injector meta.helm.sh/release-name=istiod
kubectl annotate MutatingWebhookConfiguration istio-sidecar-injector meta.helm.sh/release-namespace=istio-system

# istio-ingress
kubectl -n istio-system label ServiceAccount istio-ingressgateway-service-account app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ServiceAccount istio-ingressgateway-service-account meta.helm.sh/release-name=istio-ingressgateway
kubectl -n istio-system annotate ServiceAccount istio-ingressgateway-service-account meta.helm.sh/release-namespace=istio-system

kubectl -n istio-system label Service istio-ingressgateway app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Service istio-ingressgateway meta.helm.sh/release-name=istio-ingressgateway
kubectl -n istio-system annotate Service istio-ingressgateway meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label Deployment istio-ingressgateway app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Deployment istio-ingressgateway meta.helm.sh/release-name=istio-ingressgateway
kubectl -n istio-system annotate Deployment istio-ingressgateway meta.helm.sh/release-namespace=istio-system

kubectl -n istio-system label Role istio-ingressgateway-sds app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Role istio-ingressgateway-sds meta.helm.sh/release-name=istio-ingressgateway
kubectl -n istio-system annotate Role istio-ingressgateway-sds meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label RoleBinding istio-ingressgateway-sds app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate RoleBinding istio-ingressgateway-sds meta.helm.sh/release-name=istio-ingressgateway
kubectl -n istio-system annotate RoleBinding istio-ingressgateway-sds meta.helm.sh/release-namespace=istio-system

# istio-egress
kubectl -n istio-system label ServiceAccount istio-egressgateway-service-account app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate ServiceAccount istio-egressgateway-service-account meta.helm.sh/release-name=istio-egressgateway
kubectl -n istio-system annotate ServiceAccount istio-egressgateway-service-account meta.helm.sh/release-namespace=istio-system

kubectl -n istio-system label Service istio-egressgateway app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Service istio-egressgateway meta.helm.sh/release-name=istio-egressgateway
kubectl -n istio-system annotate Service istio-egressgateway meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label Deployment istio-egressgateway app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Deployment istio-egressgateway meta.helm.sh/release-name=istio-egressgateway
kubectl -n istio-system annotate Deployment istio-egressgateway meta.helm.sh/release-namespace=istio-system

kubectl -n istio-system label Role istio-egressgateway-sds app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate Role istio-egressgateway-sds meta.helm.sh/release-name=istio-egressgateway
kubectl -n istio-system annotate Role istio-egressgateway-sds meta.helm.sh/release-namespace=istio-system
 
kubectl -n istio-system label RoleBinding istio-egressgateway-sds app.kubernetes.io/managed-by=Helm
kubectl -n istio-system annotate RoleBinding istio-egressgateway-sds meta.helm.sh/release-name=istio-egressgateway
kubectl -n istio-system annotate RoleBinding istio-egressgateway-sds meta.helm.sh/release-namespace=istio-system
```

## Install Istio from the app-catalog

```text
ocne application install --namespace istio-system --name istio-base --release istio-base --version 1.19.9
ocne application install --namespace istio-system --name istiod --release istiod --version 1.19.9
ocne application install --namespace istio-system --name istio-ingress --release istio-ingressgateway --version 1.19.9
ocne application install --namespace istio-system --name istio-egress --release istio-egressgateway --version 1.19.9
```

## Restart all pods in the Istio mesh

Restart all pods in the Istio mesh to use the new Istio sidecar.
This can be achieved by rebooting the cluster, or by doing a rolling restart of each component within the Istio mesh.

### Manual restart of pods with Istio sidecars:

The instructions below are only for manually doing a restart of components installed by Verrazzano. One way to check if any pods are left using the older proxy is:

```text
kubectl get pods -A -o yaml | grep image: | grep proxyv2 | grep ghcr
```

```text
kubectl rollout restart deployment -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-controller
kubectl rollout status deployment -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-controller -w
 
kubectl rollout restart deployment -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-defaultbackend
kubectl rollout status deployment -n verrazzano-ingress-nginx ingress-controller-ingress-nginx-defaultbackend -w
 
kubectl rollout restart deployment -n mysql-operator mysql-operator
kubectl rollout status deployment -n mysql-operator mysql-operator -w
 
kubectl rollout restart deployment -n keycloak mysql-router
kubectl rollout status deployment -n keycloak mysql-router -w
 
kubectl rollout restart statefulset -n keycloak mysql
kubectl rollout status statefulset -n keycloak mysql -w
 
kubectl rollout restart statefulset -n keycloak keycloak
kubectl rollout status statefulset -n keycloak keycloak -w
 
kubectl rollout restart statefulset -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus
kubectl rollout status statefulset -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus -w
 
kubectl rollout restart statefulset -n verrazzano-system vmi-system-es-master
kubectl rollout status statefulset -n verrazzano-system vmi-system-es-master -w
 
kubectl rollout restart deployment -n verrazzano-system vmi-system-es-ingest
kubectl rollout status deployment -n verrazzano-system vmi-system-es-ingest -w
 
kubectl rollout restart deployment -n verrazzano-system vmi-system-osd
kubectl rollout status deployment -n verrazzano-system vmi-system-osd -w
 
kubectl rollout restart deployment -n verrazzano-system vmi-system-es-data-0
kubectl rollout status deployment -n verrazzano-system vmi-system-es-data-0 -w

kubectl rollout restart deployment -n verrazzano-system vmi-system-es-data-1
kubectl rollout status deployment -n verrazzano-system vmi-system-es-data-1 -w

kubectl rollout restart deployment -n verrazzano-system vmi-system-es-data-2
kubectl rollout status deployment -n verrazzano-system vmi-system-es-data-2 -w
 
kubectl rollout restart deployment -n verrazzano-system vmi-system-kiali
kubectl rollout status deployment -n verrazzano-system vmi-system-kiali -w
 
kubectl rollout restart deployment -n verrazzano-system vmi-system-grafana
kubectl rollout status deployment -n verrazzano-system vmi-system-grafana -w
 
kubectl rollout restart deployment -n verrazzano-system verrazzano-authproxy
kubectl rollout status deployment -n verrazzano-system verrazzano-authproxy -w
 
kubectl rollout restart daemonset -n verrazzano-system fluentd
kubectl rollout status daemonset -n verrazzano-system fluentd -w
 
kubectl rollout restart deployment -n verrazzano-system verrazzano-console
kubectl rollout status deployment -n verrazzano-system verrazzano-console -w
 
kubectl rollout restart deployment -n verrazzano-system weblogic-operator
kubectl rollout status deployment -n verrazzano-system weblogic-operator -w
```
