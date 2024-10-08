# Migrate to kube-prometheus-stack

### Version: v0.0.1-draft

Verrazzano installs the kube-prometheus-stack using a Helm chart named prometheus-operator.  As a result,
the Helm release is named prometheus-operator and all resources that use {RELEASE-NAME} in the Helm
manifests will have the name prometheus-operator. Consequently, you cannot directly upgrade to the 
version of kube-prometheus-stack that exists in the catalog.  T

In addition, the image section of the overrides file must be modified to use the correct images required
by Oracle Cloud Native Environment.

The purpose of this document is to migrate the Verrazzana kube-prometheus-stack (named prometheus-operator), 
to the catalog kube-prometheus-stack so that future upgrades can be done using the catalog.  

The steps are summarized below:
1. export the Helm user provided overrides to an overrides file
2. modify the image section of the overrides file
3. change promethues PV reclaim policy to detain
4. detach the prometheus PV
5. uninstall the prometheus-operator chart (which is really the kube-prometheus-stack)
6. install kube-state-metrics from the catalog using the override file from step 2 with PVC/PV overrides to use new PV

## Modify the overrides

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
```
## Change the Prometheus PV reclaim policy

Change reclaim policy to **Retain**.  
```text
# get the pv-name (VOLUME NAME)
PV_NAME=$(kubectl get pvc -n verrazzano-monitoring -o jsonpath='{.items[0].spec.volumeName}')
 
# patch the PV 
kubectl patch pv $PV_NAME -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

## Uninstall Verrazzano kube-prometheus-stack, node export, and delete pvc
```text
helm delete -n verrazzano-monitoring prometheus-operator  
helm delete -n verrazzano-monitoring prometheus-node-exporter
```

TODO - update ingress to use new service
TODO - fix istio authorization policy for new sa.  Check all authorization policies.
TODO - check network policies

## Install kube-prometheus-stack from the catalog
```text
ocne application install --name kube-prometheus-stack --namespace verrazzano-monitoring --values overrides.yaml
```
Wait until the prometheus server is running
```text
kubectl get pod -n verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus-0 

NAME                                            READY   STATUS    RESTARTS   AGE
prometheus-kube-prometheus-stack-prometheus-0   3/3     Running   0          90s
```

## Migrate metrics data from old PV to new PV
In this section, the Prometheus metrics will be copied from the old pv to the new pv.

First scale down Prometheus.
```text
 scale sts -n  verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus  --replicas=0
```

Create a pod YAML that mounts both pv's.
```text
cat > pod.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: shared-pvc
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

kubectl exec -it -n  verrazzano-monitoring  shared-pvc  

#remove Prometheus data from new pv
rm -fr prom-new/*

# create archiver of data from old pv
tar -cvf prom.tar -C prom-old/ .

# unpack data to new pv
tar -xvf prom.tar -C prom-new/ 

# delete the pod
kubectl delete -f pod.yaml 

# scale Prometheus up 
kubectl scale sts -n  verrazzano-monitoring prometheus-kube-prometheus-stack-prometheus  --replicas=1
```

## Fix accesss to the Prometheus server
In the section, you need to create the service used by auth-proxy to access Prometheus. They
name of that service is hard coded in the auth-proxy, so it is required.

Create the YAML file that specifies the service:
```text
cat > vz-prom-service.yaml <<EOF
apiVersion: v1
kind: Service
metadata:
  annotations:
    meta.helm.sh/release-name: kube-prometheus-stack
    meta.helm.sh/release-namespace: verrazzano-monitoring
  labels:
    app: kube-prometheus-stack-prometheus
    app.kubernetes.io/instance: kube-prometheus-stack
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/part-of: kube-prometheus-stack
    app.kubernetes.io/version: 45.25.0
    chart: kube-prometheus-stack-45.25.0
    heritage: Helm
    release: kube-prometheus-stack
    self-monitor: "true"
    verrazzano-component: prometheus-operator
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
kubectl edit authorizationpolicy -n verrazzano-monitoring   vmi-system-prometheus-authzpol
...
  - from:
    - source:
        namespaces:
        - verrazzano-monitoring
        principals:
        - cluster.local/ns/verrazzano-monitoring/sa/kube-prometheus-stack-operator
```
