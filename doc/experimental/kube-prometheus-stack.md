# Migrate to kube-prometheus-stack

### Version: v0.0.1-draft

The purpose of this document is describe how to migrate the Verrazzano kube-prometheus-stack (named prometheus-operator),
to the catalog kube-prometheus-stack so that future upgrades can be done using the catalog. This migration preserves
Prometheus metrics data and does not change any component versions.

This migration is needed because Verrazzano installs the kube-prometheus-stack using a release named prometheus-operator.  
As a result, all the related resources that use {RELEASE-NAME} in the Helm manifests will have the name prometheus-operator. 
Consequently, you cannot directly upgrade to the version of kube-prometheus-stack that exists in the catalog.

In addition, the image section of the overrides file must be modified to use the correct images required
by Oracle Cloud Native Environment.

The steps are summarized below:
1. export the Helm user-provided overrides to an overrides file
2. modify the image sections of the overrides file
3. change the Prometheus PV reclaim policy to Retain
5. uninstall the prometheus-operator chart (which is really the kube-prometheus-stack) and node exporter
6. install kube-state-metrics from the catalog using the override file from step 2 
7. scale Prometheus server down to zero replicas
8. migrate data using a pod that mounts both old and new PVs
9. create or change other resources needed for auth-proxy access
10. cleanup

***NOTE***
If you have a backup/restore process in place for Prometheus metrics, then we recommend that you back them up before
starting this migration.

## Creating the Helm overrides file

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-monitoring prometheus-operator > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image: ghcr.io/d' overrides.yaml
sed -i '/alertmanagerDefaultBaseImage:/d' overrides.yaml
sed -i '/alertmanagerDefaultBaseImageRegistry:/d' overrides.yaml
sed -i '/prometheusDefaultBaseImage:/d' overrides.yaml
sed -i '/prometheusDefaultBaseImageRegistry:/d' overrides.yaml
sed -i '/prometheusConfigReloader:/d' overrides.yaml 
sed -i '/image:/,+3d' overrides.yaml
sed -i '/thanosImage:/,+3d' overrides.yaml
sed -i '/nodeExporter:/,+2d' overrides.yaml

cat >> overrides.yaml <<EOF
nodeExporter:
  enabled: false
EOF
```
## Change the Prometheus PV reclaim policy
***NOTE*** The following instructions assume there are 2 Prometheus replicas

Change reclaim policy to **Retain**.  
```text
PV_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0 -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

PV_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-1 -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

## Uninstall Verrazzano prometheus-operator (actually kube-prometheus-stack)
```text
helm delete -n verrazzano-monitoring prometheus-operator  
```

## Install kube-prometheus-stack from the catalog
Install the kube-prometheus-stack, be sure to specify the overrides file.
```text
ocne application install --name kube-prometheus-stack --namespace verrazzano-monitoring --values overrides.yaml
```
Wait until the prometheus servers are running
```text
kubectl get pod -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus-0 
kubectl get pod -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus-1
```

## Fix access to the Prometheus server
In the section, you need to create the service used by auth-proxy to access Prometheus.
This service is required because the name of the service is hard-coded in the auth-proxy.

Create the YAML file that specifies the service:
```text
cat > vz-prom-service.yaml <<EOF
apiVersion: v1
kind: Service
metadata:
  labels:
    app: prometheus-operator-kube-p-prometheus
  name: prometheus-operator-kube-p-prometheus
  namespace: verrazzano-monitoring
spec:
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: http-web
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app.kubernetes.io/name: prometheus
    prometheus: kube-prometheus-stack-prometheus
  sessionAffinity: None
  type: ClusterIP
  EOF
```

Create the service:
```text
kubectl apply -f vz-prom-service.yaml
```

Update the AuthorizationPolicy.  This is a one line change done using kubectl edit.
Simply remove the existing principal for the verrazzano-monitoring namespace and
replace it with the new one as shown below:

```text
kubectl edit AuthorizationPolicy -n verrazzano-monitoring   vmi-system-prometheus-authzpol
```
Replace `cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus`
with `cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-operator`
as shown below in the principals section:
```text
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-operator
```

## Migrate metrics data from old PV to new PV
In this section, the Prometheus metrics will be copied from the old PV to the new PV.

First scale-in Prometheus.
```text
 scale sts -n  verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus  --replicas=0
```

### Copy data from old PV to new PV
***NOTE*** The following instructions assume there are 2 Prometheus replicas.
Repeat this section once, changing the `claimName` field by replacing. the string `prometheus-0` with `prometheus-1`.

Create a pod YAML file that mounts both PVCs.
```text
cat > pod.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: migrate-data
  namespace: verrazzano-monitoring
  labels:
    sidecar.istio.io/inject: "false"
spec:
  containers:
  - name: c1
    image: container-registry.oracle.com/os/oraclelinux:8
    command: ['sh', '-c', 'echo "The app is running!" && tail -f /dev/null']
    volumeMounts:
    - name: pvc-old
      mountPath: /prom-old
    - name: pvc-new
      mountPath: /prom-new
  volumes:    
  - name: pvc-old
    persistentVolumeClaim:
      claimName: prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0
  - name: pvc-new
    persistentVolumeClaim:
      claimName: prometheus-kube-prometheus-stack-prometheus-db-prometheus-kube-prometheus-stack-prometheus-0 
 EOF
```

Create the pod
```text
kubectl apply -f pod.yaml
```

Connect to the pod and copy the data
```text
kubectl exec -it -n  verrazzano-monitoring  migrate-data  
```

Remove Prometheus data from new PV
```text
rm -fr prom-new/*
```

Create archive of data from old PV
```text
tar -cvf prom.tar -C prom-old/ .
```

Unpack data into new PV and delete the tar file
```text
tar -xvf prom.tar -C prom-new/ 
rm prom.tar
```

Delete the pod
```text
kubectl delete -f pod.yaml
```

## Scale-out Prometheus
```text
kubectl scale sts -n  verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus --replicas=2
```

## Validate access to Grafana and Prometheus
At this point, you should be able to see your pre-migration data and new data from Grafana and Prometheus console.
Log into those consoles and ensure there is data being scraped and that you can access your pre-migration data.


## Remove old PVCs and PVs
Delete the old PVCs
```text
PV_0_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0 -o jsonpath='{.spec.volumeName}')
kubeclt delete pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0

PV_1_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-1 -o jsonpath='{.spec.volumeName}')
kubeclt delete pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-1
```

Delete the old PVs
```text
kubectl delete pv  $PV_0_NAME
kubeclt delete pv  $PV_1_NAME
```

## Summary
At this point, you should be able to see your pre-migration data and new data from Grafana and Prometheus console.
All future upgrades to kube-prometheus-stack should be done directly from the catalog as with any other component.
