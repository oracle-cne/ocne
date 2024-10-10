# Phase One: Verrazzano Migration

### Version: v0.0.3-draft

The instructions must be performed in the sequence outlined in this document.

## Install Oracle Cloud Native Environment 2.0 CLI

Follow these [instructions](https://docs.oracle.com/en/operating-systems/olcne/2.0/cli/ocne_install_task.html#ocne_install) to install the 2.0 CLI on the cluster.

## Install Oracle Cloud Native Environment 2.0 Catalog and UI

```text
ocne cluster start --provider none --kubeconfig $KUBECONFIG --auto-start-ui false
kubectl -n ocne-system rollout status deployment ocne-catalog
```

## Turn off the Verrazzano controllers

Follow these [instructions](./disable-verrazzano.md) to disable the Verrazzano controllers on the cluster.

## Upgrade to Istio 1.19.9

See [upgrade Istio](./upgrade-istio.md).

## Modify cert-manager Helm overrides

Verrazzano deployed cert-manager using Helm overrides to specify the container images.
Update the existing installation to remove those overrides,
and instead Helm will get the container image values from the defaults in the catalog.

**TBD**

## Modify WebLogic Kubernetes Operator Helm overrides

Verrazzano deployed the WebLogic Kubernetes Operator using Helm overrides to specify the container images. Update the existing installation to remove those overrides, and instead Helm will get the container images values from the defaults in the catalog.

The following example assumes WebLogic Kubernetes Operator 4.2.5 is already installed.

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
ocne application update --release weblogic-operator --namespace verrazzano-system --version 4.2.5 --catalog "WebLogic Kubernetes Operator" --reset-values --values overrides.yaml
```

## Modify Fluentd Helm overrides

Verrazzano deployed Fluentd using Helm overrides to specify the container images. Update the existing installation to remove those overrides, and instead Helm will get the container images values from the defaults in the catalog.

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-system fluentd > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/fluentdImage:/d' overrides.yaml
```

Update the existing installation:
```text
ocne application update --release fluentd --namespace verrazzano-system --version 1.14.5 --reset-values --values overrides.yaml
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

## Modify kube-prometheus-stack (named as prometheus-operator) to be managed by Helm

**TBD**  We may need to uninstall prometheus-operator and then install kube-prometheus-stack.

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