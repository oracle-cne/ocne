# Copy Prometheus Data Accross PVs

### Version: v0.0.1-draft

The purpose of this document is describe how to copy prometheus data between two PVs when 
both PVs cannot be mounted to a pod simultaneously.

## Migrate metrics data from old PV to new PV
In this section, the Prometheus metrics will be copied from the old PV to the new PV, through
your local system.

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

Create a pod YAML file that mounts both PVCs.
```text
cat > pod-old.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: migrate-data-old
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
  volumes:    
  - name: pvc-old
    persistentVolumeClaim:
      claimName: prometheus-prometheus-operator-kube-p-prometheus-db-prometheus-prometheus-operator-kube-p-prometheus-0
EOF
```

```text
cat > pod-new.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: migrate-data-new
  namespace: verrazzano-monitoring
  labels:
    sidecar.istio.io/inject: "false"
spec:
  containers:
  - name: c1
    image: container-registry.oracle.com/os/oraclelinux:8
    command: ['sh', '-c', 'echo "The app is running!" && tail -f /dev/null']
    volumeMounts:
    - name: pvc-new
      mountPath: /prom-new
  volumes:    
  - name: pvc-new
    persistentVolumeClaim:
      claimName: prometheus-kube-prometheus-stack-prometheus-db-prometheus-kube-prometheus-stack-prometheus-0 
EOF
```

Create the pods
```text
kubectl apply -f pod-old.yaml
kubectl apply -f pod-new.yaml
```

Once the pods are ready, copy the data from the old pod to your local system
```text
mkdir ./prom-data
kubectl cp --retries=5 verrazzano-monitoring/migrate-data-old:/prom-old ./prom-data
```

Copy the old data from your local system to the new pod
```text
kubectl exec -it -n verrazzano-monitoring migrate-data-new:prom-new rm -fr prom-new/*
kubectl cp --retries=5 ./prom-data verrazzano-monitoring/migrate-data-new:/prom-new
rm -fr ./prom-data
```

Delete the pods
```text
kubectl delete -f pod-old.yaml --force
kubectl delete -f pod-new.yaml --force
```
