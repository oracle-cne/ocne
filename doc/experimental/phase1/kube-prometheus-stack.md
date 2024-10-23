# Migrate to kube-prometheus-stack

### Version: v0.0.6-draft

The purpose of this document is describe how to migrate the Verrazzano kube-prometheus-stack (named prometheus-operator)
to the catalog kube-prometheus-stack. Once this is done, upgrades to kube-prometheus-stack can be done using the catalog. 
This migration preserves Prometheus metrics data and does not change any component versions.

This migration is needed because Verrazzano installs the kube-prometheus-stack using a Helm release named prometheus-operator.  
As a result, all the related resources that use {RELEASE-NAME} in the Helm manifests will have the name prometheus-operator. 
Consequently, you cannot directly upgrade to the version of kube-prometheus-stack that exists in the catalog.

In addition, the image section of the overrides file must be modified to use the correct images required
by Oracle Cloud Native Environment.

***NOTE***
These instructions assume there is one Prometheus replica.  If the replica count is different, then adjust accordingly.

The steps are summarized below:
1. Export the Helm user-provided overrides to an overrides file
2. Modify the image sections of the overrides file
3. Change the Prometheus PV reclaim policy to Retain
5. Uninstall the prometheus-operator chart (which is really the kube-prometheus-stack) and node exporter
6. Install kube-state-metrics from the catalog using the override file from step 2 
7. Scale Prometheus server down to zero replicas
8. Migrate data using a pod that mounts both old and new PVs
9. Create or change other resources needed for auth-proxy access
10. Validate metrics using Grafana
11. Cleanup

***NOTE***
If you have a backup/restore process in place for Prometheus metrics, then we recommend that you back them up before
starting this migration.

## Creating the Helm overrides file
Export the user supplied overrides of the current release to a file, 
remove the image overrides and disable nodeExporter install:
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
sed -i '/nodeExporter:/,+1d' overrides.yaml

cat >> overrides.yaml <<EOF
nodeExporter:
  enabled: false
EOF
```
## Change the Prometheus PV reclaim policy
Change reclaim policy to **Retain**.  
```text
PV_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0 -o jsonpath='{.spec.volumeName}')
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

## Uninstall Verrazzano prometheus-operator (actually kube-prometheus-stack)
```text
helm uninstall -n verrazzano-monitoring prometheus-operator --wait
```

## Install kube-prometheus-stack from the catalog
Install the kube-prometheus-stack, be sure to specify the overrides file.
```text
ocne application install --kubeconfig $KUBECONFIG --name kube-prometheus-stack --namespace verrazzano-monitoring --values overrides.yaml
```
Wait until the prometheus operator and servers are running
```text
kubectl rollout status deployment -n verrazzano-monitoring kube-prometheus-stack-operator
kubectl rollout status statefulset -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus
```

## Create service needed by auth-proxy to access the Prometheus server
In the section, you need to create the service used by auth-proxy to access Prometheus.
This service is required because the name of the service is hard-coded in the auth-proxy.

Create the YAML file that specifies the service:
```text
cat <<EOF > vz-prom-service.yaml
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

## Update AuthorizationPolicies
You need to update several AuthorizationPolicies policies.
This is a one line change to each policy done using kubectl edit.  

Replace old `cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus`  
with new `cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-prometheus`  

For example, the old principal was replaced with the new one as shown below:

```text
kubectl edit AuthorizationPolicy -n verrazzano-monitoring vmi-system-prometheus-authzpol
```
```text
      ...
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-prometheus
```

Do the same for the following AuthorizationPolicies

kubectl edit AuthorizationPolicy -n verrazzano-monitoring   vmi-system-prometheus-authzpol  

kubectl edit  AuthorizationPolicy -n verrazzano-system       verrazzano-authproxy-authzpol  

kubectl edit AuthorizationPolicy -n verrazzano-system       vmi-system-es-ingest-authzpol  

kubectl edit AuthorizationPolicy -n verrazzano-system       vmi-system-es-master-authzpol  

kubectl edit AuthorizationPolicy -n verrazzano-system       vmi-system-grafana-authzpol  

kubectl edit AuthorizationPolicy -n verrazzano-system       vmi-system-osd-authzpol  

Delete the obsolete AuthorizationPolicy
```text
kubectl delete AuthorizationPolicy -n verrazzano-system verrazzano-console-authzpol
```

### Update Application AuthorizationPolicies
Do the same for all applications that you deployed with Verrazzano.

Run this command to find the policies, then edit each and change the principal as described above:
```text
kubectl get AuthorizationPolicy -A -o yaml | grep prometheus-operator-kube-p-prometheus -B20
```
For each policy, replace old `cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus`  
with new `cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-prometheus`


## Migrate metrics data from old PV to new PV
In this section, the Prometheus metrics will be copied from the old PV to the new PV.

Scale-in Prometheus so that all pods are shutdown
```text
kubectl patch prometheus -n verrazzano-monitoring kube-prometheus-stack-prometheus --type='merge' -p '{"spec":{"replicas":0}}'
```

Make sure the stateful set has 0 pods ready
```text
kubectl get statefulset -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus
```
Results should be
```text
NAME                                          READY   ...
prometheus-kube-prometheus-stack-prometheus   0/0     ...
```


### Copy data from the old PV to new PV
***NOTE*** Repeat this section for each additional replica, changing both of the `claimName` fields by replacing the string `prometheus-0` with `prometheus-1`, for example.

***NOTE***
The pod YAML specified below will only start if the pod can mount both PVs at the same time.
If that is not the case, then the pod will not start. Use the alternate method for copying
Prometheus data as described [Copy Prometheus data via local system](../phase1/copy-prom-data-alternate.md)

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

Once the pod is ready, connect to the pod 
```text
kubectl exec -it -n verrazzano-monitoring  migrate-data bash
```

Remove the old Prometheus data from new PV
```text
rm -fr prom-new/*
```

Copy the old Prometheus data to the new PV
```text
cp -R /prom-old/. prom-new/ 
```

Delete the pod
```text
kubectl delete -f pod.yaml --force
```

## Patch ServiceMonitors and PodMonitors
Both the ServiceMonitors and PodMonitors need to be patched so that they can be processed by the Prometheus operator.

### Patch the system ServiceMonitors and PodMonitors
Patch the following objects: 
```text
kubectl patch servicemonitor -n verrazzano-monitoring authproxy --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring fluentd --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring jaeger --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring jaeger-operator --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-apiserver --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-coredns --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-kube-controller-manager --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-kube-etcd --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-kube-proxy --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-kube-scheduler --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-kubelet --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-operator --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-prometheus-stack-prometheus --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring kube-state-metrics --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring opensearch --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring pilot --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n verrazzano-monitoring prometheus-node-exporter --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'

kubectl patch podmonitor -n verrazzano-monitoring envoy-stats --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch podmonitor -n verrazzano-monitoring  nginx-ingress-controller --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
```

### Patch the application ServiceMonitors and PodMonitors
You also need to patch your application objects. For example:
```text
kubectl patch servicemonitor -n todo-list todo-appconf-todo-list-todo-domain --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
kubectl patch servicemonitor -n todo-list todo-appconf-todo-list-todo-mysql-deployment --type='merge'  -p '{"metadata":{"labels":{"release":"kube-prometheus-stack"}}}'
```

## Scale-out Prometheus
Scale-out Prometheus so that it starts all the replicas.  

***NOTE*** If you have more than one replica, then adjust the "replicas:" values below.

```text
kubectl patch prometheus -n verrazzano-monitoring kube-prometheus-stack-prometheus --type='merge' -p '{"spec":{"replicas":1}}'
kubectl rollout status statefulset -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus
```

## Validate access to Grafana and Prometheus
At this point, you should be able to see your pre-migration data and new data from Grafana and Prometheus console.
Log into those consoles and ensure there is data being scraped and that you can access your pre-migration data.


## Remove old PVC and PV
Delete the old PVC
```text
PV_NAME=$(kubectl get pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0 -o jsonpath='{.spec.volumeName}')
kubectl delete pvc -n verrazzano-monitoring prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0
```

Delete the old PV
```text
kubectl delete pv $PV_NAME
```

## Summary
At this point, you should be able to see your pre-migration data and new data from Grafana and Prometheus console.
All future upgrades to kube-prometheus-stack should be done directly from the catalog as with any other component.
