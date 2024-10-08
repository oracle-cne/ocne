# Phase One: Verrazzano Migration

### Version: v0.0.1-draft

The instructions must be performed in the sequence outlined in this document.

## Install Oracle Cloud Native Environment 2.0 CLI

Follow these [instructions](https://docs.oracle.com/en/operating-systems/olcne/2.0/cli/ocne_install_task.html#ocne_install) to install the 2.0 CLI on the cluster.

## Perform a Custer Dump

Perform a cluster dump to take snapshot of the cluster state before the migration begins.
This may take several minutes, it varies depending on the size of your cluster and number of cluster objects.
If you want to redact sensitive information, such as host names, or omit configmaps then, remove the
respective flags:

```text
ocne cluster dump --skip-redaction --include-configmaps -d /tmp/dump/before-phase1
```

## Install Oracle Cloud Native Environment 2.0 Catalog and UI

```text
ocne cluster start --provider none --kubeconfig $KUBECONFIG --auto-start-ui false
kubectl -n ocne-system rollout status deployment ocne-catalog
```

## Turn off the Verrazzano controllers

Follow these [instructions](./disable-verrazzano.md) to disable the Verrazzano controllers on the cluster.

## Modify Istio to be managed by Helm

Verrazzano does not deploy Istio using a Helm chart.
The installed version of Istio needs to be transformed to be manageable by Helm.

**TBD** - we need a set of instructions to update Istio to be managed by helm.  Verrazzano 1.6.10 installed Istio 1.19.0-1.  This version of Istio is not yet in the 2.0 application catalog.

## Modify cert-manager Helm overrides

Verrazzano deployed cert-manager using Helm overrides to specify the container images.
Update the existing installation to remove those overrides,
and instead Helm will get the container image values from the defaults in the catalog.

**TBD**

## Modify WebLogic Kubernetes Operator Helm overrides

Verrazzano deployed the WebLogic Kubernetes Operator using Helm overrides to specify the container images.  
Update the existing installation to remove those overrides, and instead Helm will get the container images values from the defaults in the catalog.

The following example assumes WebLogic Kubernetes Operator 4.1.2 is already installed.

Add the WebLogic Kubernetes Operator helm chart catalog:
```text
ocne catalog add --uri https://oracle.github.io/weblogic-kubernetes-operator --name "WebLogic Kubernetes Operator"
```

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-system weblogic-operator > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image:/d' overrides.yaml
sed -i '/weblogicMonitoringExporterImage:/d' overrides.yaml
```

Update the existing installation:
```text
ocne application update --release weblogic-operator --namespace verrazzano-system --version 4.1.2 --catalog "WebLogic Kubernetes Operator" --reset-values --values overrides.yaml
```

## Modify ingress-nginx Helm overrides

Verrazzano deployed ingress-nginx using Helm overrides to specify the container images.
Update the existing installation to remove those overrides, 
and instead Helm will get the container image values from the defaults in the catalog.

**TBD**

## Modify Grafana to be managed by Helm

Verrazzano does not deploy Grafana using a Helm chart.
The installed version of Grafana needs to be transformed to be manageable by Helm.

**TBD**

## Modify kube-prometheus-stack to be managed by Helm

See [Migrate kube-prometheus-stack](./kube-prometheus-stack.md)

## Modify prometheus-node-exporter Helm Overrides

**TBD**

## Modify kube-state-metrics to be managed by Helm

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-monitoring kube-state-metrics > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image:/,+3d' overrides.yaml
```

Update the existing installation:
```text
ocne application update --release kube-state-metrics --namespace verrazzano-monitoring --version 2.8.2 --reset-values --values overrides.yaml
```

## OAM Migration

### Generated Kubernetes Manifests
[Generated Kubernetes Manifests from OAM](./oam-to-kubernetes.md)

### Remove OAM resources
TDB

## Perform another Custer Dump

Perform a cluster dump to take snapshot of the cluster state after phase-1 is done.
This may take several minutes, it varies depending on the size of your cluster and number of cluster objects.
If you want to redact sensitive information, such as host names, or omit configmaps then, remove the
respective flags:

```text
ocne cluster dump --skip-redaction --include-configmaps -d /tmp/dump/after-phase1
```
