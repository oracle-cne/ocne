# Upgrade OpenSearch

### Version: v0.0.1-draft

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

## Install OpenSearch from the app-catalog
Convert the existing OpenSearch 2.3.0 to be managed by Helm. Deploy OpenSearch 2.15.0 from the app-catalog, however, override the images to use OpenSearch 2.3.0.  This deployment should not result in any changes to the running OpenSearch environment. However, it will cause some OpenSearch pods to restart due to minor text differences in the deployed objects.

Create environment variables for:
```text
export V8O_CONSOLE_SECRET=$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo)
export INGRESS_IP=$(kubectl get ingress -n verrazzano-system vmi-system-osd -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

Generate the values override file for the Helm deployment:
```text
envsubst > values.yaml - <<EOF
image:
  repository: ghcr.io/verrazzano/opensearch
  tag: 2.3.0-20230914055551-b6247ad8ac8
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
      /usr/local/bin/docker-entrypoint.sh
    env:
      extraEnv:
      - name: POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
esData:
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
      # Updating opensearch keystore with keys required for the repository-s3 plugin
      if [ \${OBJECT_STORE_ACCESS_KEY_ID:-} ]; then
        echo Updating object store access key...
        echo \$OBJECT_STORE_ACCESS_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.access_key;
      fi
      if [ \${OBJECT_STORE_SECRET_KEY_ID:-} ]; then
        echo Updating object store secret key...
        echo \$OBJECT_STORE_SECRET_KEY_ID | /usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.secret_key;
      fi
      # Disable the jvm heap settings in jvm.options
      echo Commenting out java heap settings in jvm.options...
      sed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options
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

## Start Rolling Upgrade to OpenSearch 2.15.0
These instructions are based on the rolling upgrade instructions in the OpenSearch documentation.

**Check Cluster Health**

A status of green indicates that all primary and replica shards are allocated.
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/health?pretty
```

## Disable Shard Replication

Disable shard replication to prevent shard replicas from being created while nodes are being taken offline. This stops the movement of Lucene index segments on nodes in your cluster.

```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} -X PUT -H "Content-Type: application/json"  https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/settings?pretty -d '
{
    "persistent": {
    "cluster.routing.allocation.enable": "primaries"
    }
}'
```

**Query settings**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/settings?pretty
```

**Flush**

Perform a flush operation on the cluster to commit transaction log entries to the Lucene index.
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} -X POST https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_flush?pretty
```

**Identify which node was promoted to cluster manager**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cat/nodes?v&h=name,version,node.role,master | column -t
```

## Update each of the data nodes
Review your cluster and identify the first node to upgrade. Eligible cluster manager nodes should be upgraded last because OpenSearch nodes can join a cluster with manager nodes running an older version, but they cannot join a cluster with all manager nodes running a newer version.

**Scale data node deployment to 0**
```text
kubectl scale deployment -n verrazzano-system vmi-system-es-data-0 --replicas 0
```

**Confirm the node has been dismissed from the cluster**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cat/nodes?v&h=name,version,node.role,master | column -t
```

**Patch the deployment to use OpenSearch 2.15.0 container**

Update the OpenSearch image to 2.15.0
```text
kubectl patch deployment --namespace verrazzano-system vmi-system-es-data-0 -p '{"spec": {"template": {"spec": {"containers":[{"name": "es-data", "image": "olcne/opensearch:v2.15.0"}]}}}}' --type=strategic
```

Update the container command
```text
kubectl patch deployment --namespace verrazzano-system vmi-system-es-data-0 -p '{"spec": {"template": {"spec": {"containers":[{"name": "es-data", "command": ["sh", "-c", "#!/usr/bin/env bash -e\n# Disable the jvm heap settings in jvm.options\necho Commenting out java heap settings in jvm.options...\nsed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options\necho network.publish_host: ${POD_IP} \u003e\u003e config/opensearch.yml\necho network.bind_host: 0.0.0.0 \u003e\u003e config/opensearch.yml\necho cluster.name: $(printenv cluster.name) \u003e\u003e config/opensearch.yml\necho node.attr.availability_domain: $(node.attr.availability_domain) \u003e\u003e config/opensearch.yml\necho node.roles: $(printenv node.roles) \u003e\u003e config/opensearch.yml\necho discovery.seed_hosts: $(printenv discovery.seed_hosts) \u003e\u003e config/opensearch.yml\necho logger.org.opensearch: $(printenv logger.org.opensearch) \u003e\u003e config/opensearch.yml\n/usr/local/bin/docker-entrypoint.sh"]}]}}}}' --type=strategic
```

**Scale deployment back to 1**
```text
kubectl scale deployment -n verrazzano-system vmi-system-es-data-0 --replicas 1
```

**Confirm that the node has rejoined the cluster**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cat/nodes?v&h=name,version,node.role,master | column -t
```

**Repeat the above steps for the deployments vmi-system-es-data-1 and vmi-system-es-data-2**

## Update the es-ingest node
Perform the similar set of steps to the es-ingest node as for updating the es-data nodes.

**Scale ingest node deployment to 0**
```text
kubectl scale deployment -n verrazzano-system vmi-system-es-ingest --replicas 0
```

**Patch the deployment to use OpenSearch 2.15.0 container**

Update the OpenSearch image to 2.15.0
```text
kubectl patch deployment --namespace verrazzano-system vmi-system-es-ingest -p '{"spec": {"template": {"spec": {"containers":[{"name": "es-ingest", "image": "olcne/opensearch:v2.15.0"}]}}}}' --type=strategic
```

Update the container command
```text
kubectl patch deployment --namespace verrazzano-system vmi-system-es-ingest -p '{"spec": {"template": {"spec": {"containers":[{"name": "es-ingest", "command": ["sh", "-c", "#!/usr/bin/env bash -e\nset -euo pipefail\necho network.publish_host: ${POD_IP} \u003e\u003e config/opensearch.yml\necho network.bind_host: 0.0.0.0 \u003e\u003e config/opensearch.yml\necho cluster.name: $(printenv cluster.name) \u003e\u003e config/opensearch.yml\necho logger.org.opensearch: $(printenv logger.org.opensearch) \u003e\u003e config/opensearch.yml\necho discovery.seed_hosts: $(printenv discovery.seed_hosts) \u003e\u003e config/opensearch.yml\necho node.roles: $(printenv node.roles) \u003e\u003e config/opensearch.yml\n/usr/local/bin/docker-entrypoint.sh"]}]}}}}' --type=strategic
```

**Scale deployment back to 1**
```text
kubectl scale deployment -n verrazzano-system vmi-system-es-ingest --replicas 1
```

**Confirm that the node has rejoined the cluster**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cat/nodes?v&h=name,version,node.role,master | column -t
```

## Update the es-master nodes

**Patch the StatefulSet to use OpenSearch 2.15.0 container**

Update the OpenSearch image to 2.15.0
```text
kubectl patch statefulset --namespace verrazzano-system vmi-system-es-master -p '{"spec": {"template": {"spec": {"containers":[{"name": "es-master", "image": "olcne/opensearch:v2.15.0", "command": ["sh", "-c", "#!/usr/bin/env bash -e\n# Disable the jvm heap settings in jvm.options\necho Commenting out java heap settings in jvm.options...\nsed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options\necho network.publish_host: ${POD_IP} \u003e\u003e config/opensearch.yml\necho network.bind_host: 0.0.0.0 \u003e\u003e config/opensearch.yml\necho cluster.name: $(printenv cluster.name) \u003e\u003e config/opensearch.yml\necho cluster.initial_master_nodes: $(cluster.initial_master_nodes) \u003e\u003e config/opensearch.yml\necho node.roles: $(printenv node.roles) \u003e\u003e config/opensearch.yml\necho discovery.seed_hosts: $(printenv discovery.seed_hosts) \u003e\u003e config/opensearch.yml\necho logger.org.opensearch: $(printenv logger.org.opensearch) \u003e\u003e config/opensearch.yml\n/usr/local/bin/docker-entrypoint.sh"]}]}}}}' --type=strategic
```

**Confirm that the node has rejoined the cluster**
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cat/nodes?v&h=name,version,node.role,master | column -t
```

## Re-enable shard replication
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} -X PUT -H "Content-Type: application/json"  https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/settings?pretty -d '
{
    "persistent": {
        "cluster.routing.allocation.enable": "all"
    }
}'
```

## Check Cluster Health
A status of green indicates that all primary and replica shards are allocated.
```text
curl -k -u verrazzano:${V8O_CONSOLE_SECRET} https://opensearch.vmi.system.default.${INGRESS_IP}.nip.io/_cluster/health?pretty
```


## Update Helm Deployment
Update the Helm deployment to contain all the patches that were applied during the rolling upgrade. This step should have no effect on the running system, the pods may not even restart.

```text
ocne app install --name opensearch --release opensearch --namespace verrazzano-system --catalog embedded --version 2.15.0 --values - <<EOF
image:
  repository: ghcr.io/verrazzano/opensearch
  tag: 2.3.0-20230914055551-b6247ad8ac8
esMaster:
  esMaster:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
      # Disable the jvm heap settings in jvm.options
      echo Commenting out java heap settings in jvm.options...
      sed -i -e /^-Xms/s/^/#/g -e /^-Xmx/s/^/#/g config/jvm.options
      echo transport.publish_host: \${POD_IP} >> config/opensearch.yml
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
esIngest:
esData:
EOF
```




## Later - need to update es-master to 2.15.0 settings



```text
esMaster:
  esMaster:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
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
```

```text
esData:
  esData:
    command:
    - sh
    - -c
    - |-
      #!/usr/bin/env bash -e
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
```

```text
esIngest:
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
```