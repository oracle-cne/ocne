# Upgrade Grafana

### Version: v0.0.1-draft

Upgrade from Grafana 7.5.15 to 7.5.17.

## Change the Grafana PV reclaim policy
Change reclaim policy to **Retain**.
```text
PV_NAME=$(kubectl get pvc -n verrazzano-system vmi-system-grafana -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```
## Export a copy of the verrazzano-grafana-dashboards deployment
The Grafana dashboards packaged with Verrazzano were deployed using an internal Verrazzano Helm chart. We recommend exporting a copy of the Helm manfest for backup.
```text
helm get manifest verrazzano-grafana-dashboards -n verrazzano-system > verrazzano-grafana-dashboards.manifest
```

## Delete the Grafana deployment
The Grafana deployment needs to be deleted because the upgrade will fail due to the `matchLabels` content being different in the Helm chart.  The `matchLables` are an immutable field, therefore the deployment needs to be deleted and re-created.

```text
kubectl delete deployment vmi-system-grafana --namespace verrazzano-system
```

## Modify the Grafana objects to be annotated as being managed by Helm

Verrazzano does not deploy Grafana using a Helm chart.
The installed version of Grafana needs to be transformed to be manageable by Helm.

```text
kubectl -n verrazzano-system label ServiceAccount verrazzano-monitoring-operator app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate ServiceAccount verrazzano-monitoring-operator meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate ServiceAccount verrazzano-monitoring-operator meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label Service vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate Service vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate Service vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label Ingress vmi-system-grafana app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate Ingress vmi-system-grafana meta.helm.sh/release-name=vmi-system-grafana
kubectl -n verrazzano-system annotate Ingress vmi-system-grafana meta.helm.sh/release-namespace=verrazzano-system
```

## Extract configuration information
Extract some configuration information that will be required in the Helm overrides:
```text
GRAFANA_HOST=$(kubectl get ingress -n verrazzano-system vmi-system-grafana -o jsonpath='{.spec.rules[0].host}')
```

## Create Helm Overrides File

Generate a Helm overrides file, based on Verrazzano's default install of Grafana:

```text
cat > overrides.yaml <<EOF
nameOverride: system-grafana
podLabels:
  app: system-grafana
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
grafana.ini:
  server:
    domain: ${GRAFANA_HOST}
    root_url: https://${GRAFANA_HOST}
    enable_gzip: true
  users:
    allow_sign_up: false
    auto_assign_org: true
    auto_assign_org_role: Viewer
  auth:
    disable_login_form: true
    disable_signout_menu: true
  auth.basic:
    enabled: false
  auth.anonymous:
    enabled: false
  auth.proxy:
    enabled: true
    header_name: X-WEBAUTH-USER
    header_property: username
    auto_sign_up: true
env:
  PROMETHEUS_TARGETS: http://vmi-system-prometheus:9090
livenessProbe:
  failureThreshold: 3
  httpGet:
    path: /api/health
    port: 3000
    scheme: HTTP
  initialDelaySeconds: 15
  periodSeconds: 20
  successThreshold: 1
  timeoutSeconds: 3
readinessProbe:
  failureThreshold: 3
  httpGet:
    path: /api/health
    port: 3000
    scheme: HTTP
  initialDelaySeconds: 15
  periodSeconds: 20
  successThreshold: 1
  timeoutSeconds: 3
resources:
  requests:
    memory: 48Mi
containerSecurityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
  privileged: false
  runAsGroup: 472
  runAsNonRoot: true
  runAsUser: 472
serviceAccount:
  name: verrazzano-monitoring-operator
securityContext:
  seccompProfile:
    type: RuntimeDefault
persistence:
  enabled: true
  existingClaim: vmi-system-grafana
extraVolumeMounts:
  - name: dashboards-volume
    mountPath: /etc/grafana/provisioning/dashboardjson
extraConfigmapMounts:
  - name: datasources-volume
    mountPath: /etc/grafana/provisioning/datasources
    configMap: vmi-system-datasource
  - name: dashboards-provider-volume
    mountPath: /etc/grafana/provisioning/dashboards
    configMap: verrazzano-dashboard-provider
extraContainers: |-
  - name: k8s-sidecar
    env:
    - name: LABEL
      value: grafana_dashboard
    - name: LABEL_VALUE
      value: "1"
    - name: FOLDER
      value: /etc/grafana/provisioning/dashboardjson
    - name: NAMESPACE
      value: ALL
    image: ghcr.io/verrazzano/k8s-sidecar:v1.15.0-20230922083013-7adaf012
    imagePullPolicy: IfNotPresent
    resources: {}
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      privileged: false
      runAsGroup: 65534
      runAsNonRoot: true
      runAsUser: 65534
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /etc/grafana/provisioning/dashboardjson
      name: dashboards-volume
service:
  port: 8775
ingress:
  enabled: true
  ingressClassName: verrazzano-nginx
  path: /ignore
  hosts: 
    - ${GRAFANA_HOST}
  tls:
  - hosts:
    - ${GRAFANA_HOST}
    secretName: system-tls-grafana
  extraPaths:
    - path: /()(.*)
      pathType: ImplementationSpecific
      backend:
        service:
          name: verrazzano-authproxy
          port:
            number: 8775
initChownData:
  image:
    sha:
    repository: olcne/grafana
    tag: v7.5.17
rbac:
  extraClusterRoleRules:
  - apiGroups:
    - ""
    resources:
    - configmaps
    verbs:
    - watch
EOF
```

## Install Grafana 7.5.17:

```text
ocne application install --release vmi-system-grafana --name grafana --namespace verrazzano-system --version 7.5.17 --values overrides.yaml
```

Wait for the update to complete:

```text
kubectl rollout status deployment --namespace verrazzano-system vmi-system-grafana -w
```

