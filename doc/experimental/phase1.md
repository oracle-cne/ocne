# Phase One: Verrazzano Migration

### Version: v0.0.7-draft

The instructions must be performed in the sequence outlined in this document.

## Install Oracle Cloud Native Environment 2.0 CLI

Follow these [instructions](https://docs.oracle.com/en/operating-systems/olcne/2.0/cli/ocne_install_task.html#ocne_install) to install the 2.0 CLI on the cluster.

## Install Oracle Cloud Native Environment 2.0 Catalog and UI

```text
ocne cluster start --provider none --kubeconfig $KUBECONFIG --auto-start-ui false
kubectl -n ocne-system rollout status deployment ocne-catalog
```
## Perform a Cluster Dump

Perform a cluster dump to take snapshot of the cluster state before the migration begins.
This may take several minutes, it varies depending on the size of your cluster and number of cluster objects.
If you want to redact sensitive information, such as host names, or omit configmaps, then remove the
respective flags:

```text
ocne cluster dump --kubeconfig $KUBECONFIG --skip-redaction --include-configmaps -d /tmp/dump/before-phase1
```

## Turn off the Verrazzano controllers

Follow these [instructions](./disable-verrazzano.md) to disable the Verrazzano controllers on the cluster.

## Upgrade to Istio 1.19.9

See [upgrade Istio](./upgrade-istio.md).

## Modify cert-manager Helm overrides

Verrazzano deployed cert-manager using Helm overrides to specify the container images.
Update the existing installation to remove those overrides,
and instead Helm will get the container image values from the defaults in the catalog.

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n cert-manager cert-manager > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image:/,+2d' overrides.yaml
sed -i 's,ghcr.io/verrazzano/cert-manager-acmesolver:v1.9.1-20240724165802-4c06aea1,olcne/cert-manager-acmesolver:v1.9.1,' overrides.yaml
sed -i '1i installCRDs: false' overrides.yaml
```

Update the existing installation:
```text
ocne application update --release cert-manager --namespace cert-manager --version 1.9.1 --reset-values --values overrides.yaml
```

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

## Upgrade ingress-nginx from 1.7.1 to 1.9.6

Verrazzano deployed ingress-nginx using Helm overrides to specify the container images.
Update the existing installation to remove those overrides, 
and instead Helm will get the container image values from the defaults in the catalog.

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-ingress-nginx ingress-controller > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image:/,+2d' overrides.yaml
```

Uninstall prometheus-node-exporter 1.3.1. This is required because the 1.6.1 helm chart contains a different value for `spec.selector.matchLabels`, which Kubernetes rejects as an immutable field.

```text
ocne application uninstall --release prometheus-node-exporter --namespace verrazzano-monitoring
```

Install ingress-nginx 1.9.6 using the overrides extracted above:
```text
ocne application update --release ingress-controller --namespace verrazzano-ingress-nginx --version 1.9.6 --reset-values --values overrides.yaml
```

## Modify Grafana to be managed by Helm

Verrazzano does not deploy Grafana using a Helm chart.
The installed version of Grafana needs to be transformed to be manageable by Helm.

**TBD**

## Modify kube-prometheus-stack to be managed by Helm

See [Migrate kube-prometheus-stack](./kube-prometheus-stack.md)

## Upgrade prometheus-node-exporter from 1.3.1 to to 1.6.1

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-monitoring prometheus-node-exporter > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image:/,+2d' overrides.yaml
```

Uninstall prometheus-node-exporter 1.3.1. This is required because the 1.6.1 helm chart contains a different value for `spec.selector.matchLabels`, which Kubernetes rejects as an immutable field.

```text
ocne application uninstall --release prometheus-node-exporter --namespace verrazzano-monitoring
```

Install prometheus-node-exporter 1.6.1 using the overrides extracted above:
```text
ocne application install --release prometheus-node-exporter --name prometheus-node-exporter --namespace verrazzano-monitoring --version 1.6.1 --values overrides.yaml
```

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
Because OAM will no longer be used, you need to generate Kubernetes manifest YAML files for
the Kubernetes resources running in the cluster, that were generated by the Verrazzano controllers 
as a result of processing OAM resources. This must be done before you can remove OAM resources, as described in the next section.

[Generated Kubernetes Manifests from OAM](./oam-to-kubernetes.md)

### Remove OAM resources
**TBD**

## Delete the Verrazzano custom resource

**TBD**

## Perform another Cluster Dump

Perform a cluster dump to take snapshot of the cluster state after phase-1 is done.
This may take several minutes, it varies depending on the size of your cluster and number of cluster objects.
If you want to redact sensitive information, such as host names, or omit configmaps, then remove the
respective flags:

```text
ocne cluster dump --kubeconfig $KUBECONFIG --skip-redaction --include-configmaps -d /tmp/dump/after-phase1
```
