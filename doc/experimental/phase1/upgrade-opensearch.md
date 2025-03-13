# Upgrade OpenSearch

### Version: v0.0.2-draft

Upgrade from OpenSearch 2.3.0 to 2.15.0.

## Modify the OpenSearch objects to be annotated as being managed by Helm
Verrazzano does not deploy OpenSearch using a Helm chart.
The installed version of OpenSearch needs to be transformed to be manageable by Helm.

```text
kubectl -n verrazzano-system label statefulset vmi-system-es-master app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate statefulset vmi-system-es-master meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate statefulset vmi-system-es-master meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label deployment vmi-system-es-data-0 app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-0 meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-0 meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label deployment vmi-system-es-data-1 app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-1 meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-1 meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label deployment vmi-system-es-data-2 app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-2 meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate deployment vmi-system-es-data-2 meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label deployment vmi-system-es-ingest app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate deployment vmi-system-es-ingest meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate deployment vmi-system-es-ingest meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label ingress vmi-system-os-ingest app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate ingress vmi-system-os-ingest meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate ingress vmi-system-os-ingest meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label service vmi-system-es-data app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate service vmi-system-es-data meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate service vmi-system-es-data meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label service vmi-system-es-master app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate service vmi-system-es-master meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate service vmi-system-es-master meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label service vmi-system-os-ingest app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate service vmi-system-os-ingest meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate service vmi-system-os-ingest meta.helm.sh/release-namespace=verrazzano-system

kubectl -n verrazzano-system label service vmi-system-es-master-http app.kubernetes.io/managed-by=Helm
kubectl -n verrazzano-system annotate service vmi-system-es-master-http meta.helm.sh/release-name=opensearch
kubectl -n verrazzano-system annotate service vmi-system-es-master-http meta.helm.sh/release-namespace=verrazzano-system
```

## Change the OpenSearch PV reclaim policy
Change the reclaim policy to **Retain**.

```text
PV_NAME=$(kubectl get pvc -n verrazzano-system elasticsearch-master-vmi-system-es-master-0  -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-system elasticsearch-master-vmi-system-es-master-1  -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-system elasticsearch-master-vmi-system-es-master-2  -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-system vmi-system-es-data -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-system vmi-system-es-data-1 -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-system vmi-system-es-data-2 -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

## Create environment variables
Set some environment variables to be used in the upgrade steps.

```text
export V8O_CONSOLE_SECRET=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo)
export INGRESS_IP=$(kubectl get ingress -n verrazzano-system vmi-system-osd -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

## Shutdown OpenSearch

```text
kubectl scale deployment -n verrazzano-system vmi-system-es-data-0 --replicas 0
kubectl scale deployment -n verrazzano-system vmi-system-es-data-1 --replicas 0
kubectl scale deployment -n verrazzano-system vmi-system-es-data-2 --replicas 0
kubectl scale deployment -n verrazzano-system vmi-system-es-ingest --replicas 0
kubectl scale statefulset -n verrazzano-system vmi-system-es-master --replicas 0
kubectl scale deployment -n verrazzano-system vmi-system-osd --replicas 0
```

## Generate the values override file for the Helm deployment
```text
envsubst > values.yaml - <<EOF
fullnameOverride: vmi-system
ingress:
  host: opensearch.vmi.system.default.$INGRESS_IP.nip.io
  extraLabels:
    k8s-app: verrazzano.io
    vmo.v1.verrazzano.io: system
  extraAnnotations:
    cert-manager.io/cluster-issuer: verrazzano-cluster-issuer
    cert-manager.io/common-name: opensearch.vmi.system.default.$INGRESS_IP.nip.io
    external-dns.alpha.kubernetes.io/target: verrazzano-ingress.default.$INGRESS_IP.nip.io
    external-dns.alpha.kubernetes.io/ttl: "60"
    kubernetes.io/tls-acme: "true"
  spec:
    ingressClassName: verrazzano-nginx
    backend:
      service:
        name: verrazzano-authproxy
esMaster:
  serviceAccount: verrazzano-monitoring-operator
  namespace: verrazzano-system
  extraMetadataLabels:
    k8s-app: verrazzano.io
    verrazzano-component: opensearch
    vmo.v1.verrazzano.io: system
  extraTemplateLabels:
    opensearch.verrazzano.io/role-master: "true"
    verrazzano-component: opensearch
  esMaster:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
      # Updating opensearch keystore with keys
      # required for the repository-s3 plugin
      if [ \${OBJECT_STORE_ACCESS_KEY_ID:-} ]; then
        echo Updating object store access key...
        echo \$OBJECT_STORE_ACCESS_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.access_key;
      fi
      if [ \${OBJECT_STORE_SECRET_KEY_ID:-} ]; then
        echo Updating objectstore secret key...
        echo \$OBJECT_STORE_SECRET_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.secret_key;
      fi
      # Disable the jvm heap settings in jvm.options
      echo Commenting out java heap settings in jvm.options...
      sed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options
      echo network.publish_host: \${POD_IP} >> config/opensearch.yml
      echo network.bind_host: 0.0.0.0 >> config/opensearch.yml
      echo cluster.name: \$(printenv cluster.name) >> config/opensearch.yml
      echo cluster.initial_master_nodes: \$(cluster.initial_master_nodes) >> config/opensearch.yml
      echo node.roles: \$(printenv node.roles) >> config/opensearch.yml
      echo discovery.seed_hosts: \$(printenv discovery.seed_hosts) >> config/opensearch.yml
      echo logger.org.opensearch: \$(printenv logger.org.opensearch) >> config/opensearch.yml
      /usr/local/bin/docker-entrypoint.sh
    env:
      extraEnv:
      - name: OBJECT_STORE_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            key: object_store_access_key
            name: verrazzano-backup
            optional: true
      - name: OBJECT_STORE_SECRET_KEY_ID
        valueFrom:
          secretKeyRef:
            key: object_store_secret_key
            name: verrazzano-backup
            optional: true
      - name: POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
  service:
    extraLabels:
      k8s-app: verrazzano.io
      verrazzano-component: opensearch
      vmo.v1.verrazzano.io: system
    extraSelectorLabels:
      opensearch.verrazzano.io/role-master: "true"
esIngest:
  serviceAccount: verrazzano-monitoring-operator
  extraLabels:
    k8s-app: verrazzano.io
    opensearch.verrazzano.io/role-ingest: "true"
    verrazzano-component: opensearch
    vmo.v1.verrazzano.io: system
  extraTemplateLabels:
    opensearch.verrazzano.io/role-ingest: "true"
    verrazzano-component: opensearch
  esIngest:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
      set -euo pipefail
      echo network.publish_host: \${POD_IP} >> config/opensearch.yml
      echo network.bind_host: 0.0.0.0 >> config/opensearch.yml
      echo cluster.name: \$(printenv cluster.name) >> config/opensearch.yml
      echo logger.org.opensearch: \$(printenv logger.org.opensearch) >> config/opensearch.yml
      echo discovery.seed_hosts: \$(printenv discovery.seed_hosts) >> config/opensearch.yml
      echo node.roles: \$(printenv node.roles) >> config/opensearch.yml
      /usr/local/bin/docker-entrypoint.sh
    env:
      extraEnv:
      - name: POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
esData:
  deploymentSuffixes:
  - "0"
  - "1"
  - "2"
  serviceAccount: verrazzano-monitoring-operator
  service:
    extraLabels:
      k8s-app: verrazzano.io
      verrazzano-component: opensearch
      vmo.v1.verrazzano.io: system
    extraSelectorLabels:
      opensearch.verrazzano.io/role-data: "true"
  extraLabels:
    k8s-app: verrazzano.io
    opensearch.verrazzano.io/role-data: "true"
    verrazzano-component: opensearch
    vmo.v1.verrazzano.io: system
  extraTemplateLabels:
    opensearch.verrazzano.io/role-data: "true"
    verrazzano-component: opensearch
  esData:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
      # Updating opensearch keystore with keys
      # required for the repository-s3 plugin
      if [ \${OBJECT_STORE_ACCESS_KEY_ID:-} ]; then
        echo Updating object store access key...
        echo \$OBJECT_STORE_ACCESS_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.access_key;
      fi
      if [ \${OBJECT_STORE_SECRET_KEY_ID:-} ]; then
        echo Updating objectstore secret key...
        echo \$OBJECT_STORE_SECRET_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.secret_key;
      fi
      # Disable the jvm heap settings in jvm.options
      echo Commenting out java heap settings in jvm.options...
      sed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options
      echo network.publish_host: \${POD_IP} >> config/opensearch.yml
      echo network.bind_host: 0.0.0.0 >> config/opensearch.yml
      echo cluster.name: \$(printenv cluster.name) >> config/opensearch.yml
      echo node.attr.availability_domain: \$(node.attr.availability_domain) >> config/opensearch.yml
      echo node.roles: \$(printenv node.roles) >> config/opensearch.yml
      echo discovery.seed_hosts: \$(printenv discovery.seed_hosts) >> config/opensearch.yml
      echo logger.org.opensearch: \$(printenv logger.org.opensearch) >> config/opensearch.yml
      /usr/local/bin/docker-entrypoint.sh
    env:
      extraEnv:
      - name: OBJECT_STORE_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            key: object_store_access_key
            name: verrazzano-backup
            optional: true
      - name: OBJECT_STORE_SECRET_KEY_ID
        valueFrom:
          secretKeyRef:
            key: object_store_secret_key
            name: verrazzano-backup
            optional: true
      - name: POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
esMasterHttp:
  service:
    extraLabels:
      k8s-app: verrazzano.io
      verrazzano-component: opensearch
      vmo.v1.verrazzano.io: system
    extraSelectorLabels:
      opensearch.verrazzano.io/role-master: "true"
osIngest:
  service:
    extraLabels:
      k8s-app: verrazzano.io
      verrazzano-component: opensearch
      vmo.v1.verrazzano.io: system
    extraSelectorLabels:
      opensearch.verrazzano.io/role-ingest: "true"
EOF
```

Install the Helm chart.
```text
ocne app install --name opensearch --release opensearch --namespace verrazzano-system --catalog embedded --version 2.15.0 --values values.yaml
```

Wait for OpenSearch to Restart
```text
kubectl rollout status -n verrazzano-system deployment vmi-system-es-data-0 -w
kubectl rollout status -n verrazzano-system deployment vmi-system-es-data-1 -w
kubectl rollout status -n verrazzano-system deployment vmi-system-es-data-2 -w
kubectl rollout status -n verrazzano-system deployment vmi-system-es-ingest -w
kubectl rollout status -n verrazzano-system statefulset vmi-system-es-master -w
```

Re-enable OpenSearch Dashboards
```text
kubectl scale deployment -n verrazzano-system vmi-system-osd --replicas 1
kubectl rollout status -n verrazzano-system deployment vmi-system-osd -w
```

## Check Cluster Health
A status of green indicates that all primary and replica shards are allocated.
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/health?pretty
```
