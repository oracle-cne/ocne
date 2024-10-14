# Upgrade Grafana

### Version: v0.0.1-draft

Upgrade from Grafana 7.5.15 to 7.5.17.

## Modify the Grafana objects to be annotated as being managed by Helm

Verrazzano does not deploy Grafana using a Helm chart.
The installed version of Grafana needs to be transformed to be manageable by Helm.

```text
kubectl -n verrazzano-system label ServiceAccount verrazzano-monitoring-operator app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate ServiceAccount verrazzano-monitoring-operator meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate ServiceAccount verrazzano-monitoring-operator meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label Deployment vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate Deployment vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate Deployment vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label Service vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate Service vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate Service vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label Ingress vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate Ingress vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate Ingress vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label pvc vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate pvc vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate pvc vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label secret grafana-admin app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate secret grafana-admin meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate secret grafana-admin meta.helm.sh/release-namespace=verrazzano-system
```

## Create Helm Overrides File

Generate a Helm overrides file, based on Verrazzano's default install of Grafana:

```text
GRAFANA_HOST=$(kubectl get ingress -n verrazzano-system vmi-system-grafana -o jsonpath='{.spec.rules[0].host}')
cat > overrides.yaml <<EOF
nameOverride: system-grafana
deploymentStrategy:
  type: Recreate
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - podAffinityTerm:
        labelSelector:
          matchLabels:
            app: system-grafana
        topologyKey: failure-domain.beta.kubernetes.io/zone
      weight: 100
admin:
  existingSecret: grafana-admin
  userKey: username
  passwordKey: password
env:
  GF_SERVER_ENABLE_GZIP: true
  PROMETHEUS_TARGETS: http://vmi-system-prometheus:9090
  GF_AUTH_ANONYMOUS_ENABLED: false
  GF_AUTH_BASIC_ENABLED: false
  GF_USERS_ALLOW_SIGN_UP: false
  GF_USERS_AUTO_ASSIGN_ORG: true
  GF_USERS_AUTO_ASSIGN_ORG_ROLE: Viewer
  GF_AUTH_DISABLE_LOGIN_FORM: true
  GF_AUTH_DISABLE_SIGNOUT_MENU: true
  GF_AUTH_PROXY_ENABLED: true
  GF_AUTH_PROXY_HEADER_NAME: X-WEBAUTH-USER
  GF_AUTH_PROXY_HEADER_PROPERTY: username
  GF_AUTH_PROXY_AUTO_SIGN_UP: true
  GF_SERVER_DOMAIN: ${GRAFANA_HOST}
  GF_SERVER_ROOT_URL: https:/${GRAFANA_HOST}
EOF
```

## Install Grafana 7.5.17:

```text
ocne application install --release vmi-system-grafana --name grafana --namespace verrazzano-system --version 7.5.17 --values overrides.yaml
```
