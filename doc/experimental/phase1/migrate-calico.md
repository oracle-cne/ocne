# Migrate Calico

### Version: v0.0.1-draft

Migrate Calico 3.27.5 install to be managed by Helm

## Modify the Tigera Operator objects to be annotated as being managed by Helm

1.x Calico default cni installation through Kubernetes module is not deployed using a Helm Chart.
The installed version of Calico needs to be transformed to be manageable by Helm.

tigera-operator:
```text
kubectl -n tigera-operator label ServiceAccount tigera-operator app.kubernetes.io/managed-by=Helm
kubectl -n tigera-operator annotate ServiceAccount tigera-operator meta.helm.sh/release-name=mycalico
kubectl -n tigera-operator annotate ServiceAccount tigera-operator meta.helm.sh/release-namespace=tigera-operator

kubectl -n tigera-operator label ClusterRole tigera-operator app.kubernetes.io/managed-by=Helm
kubectl -n tigera-operator annotate ClusterRole tigera-operator meta.helm.sh/release-name=mycalico
kubectl -n tigera-operator annotate ClusterRole tigera-operator meta.helm.sh/release-namespace=tigera-operator

kubectl -n tigera-operator label ClusterRoleBinding tigera-operator app.kubernetes.io/managed-by=Helm
kubectl -n tigera-operator annotate ClusterRoleBinding tigera-operator meta.helm.sh/release-name=mycalico
kubectl -n tigera-operator annotate ClusterRoleBinding tigera-operator meta.helm.sh/release-namespace=tigera-operator

kubectl -n tigera-operator label Deployment tigera-operator app.kubernetes.io/managed-by=Helm
kubectl -n tigera-operator annotate Deployment tigera-operator meta.helm.sh/release-name=mycalico
kubectl -n tigera-operator annotate Deployment tigera-operator meta.helm.sh/release-namespace=tigera-operator

kubectl label APIServer default app.kubernetes.io/managed-by=Helm
kubectl annotate APIServer default meta.helm.sh/release-name=mycalico
kubectl annotate APIServer default meta.helm.sh/release-namespace=tigera-operator

kubectl label Installation default app.kubernetes.io/managed-by=Helm
kubectl annotate Installation default meta.helm.sh/release-name=mycalico
kubectl annotate Installation default meta.helm.sh/release-namespace=tigera-operator
```

```text
ocne application install --catalog embedded --name tigera-operator --version 1.32.12 --namespace tigera-operator --release mycalico
```

### Verify that the tigera-operator and all of the calico pods are running 

```text
kubectl get pods -A  | grep tigera-operator
kubectl get pods -A  | grep calico
```
