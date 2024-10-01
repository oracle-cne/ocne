# Phase One: Verrazzano Migration

### Version: v0.0.1-draft

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

## Modify Istio to be managed by Helm

Verrazzano does not deploy Istio using a Helm chart.
The installed version of Istio needs to be transformed to be manageable by Helm.

**TBD** - we need a set of instructions to update Istio to be managed by helm.  Verrazzano 1.6.10 installed Istio 1.19.0-1.  This version of Istio is not yet in the 2.0 application catalog.

## Modify WebLogic Kubernetes Operator Helm Overrides

Verrazzano deployed the WebLogic Kubernetes Operator using Helm overrides to specify the container images.  
Update the existing installation to remove those overrides and let Helm get the container images values from the defaults in the catalog.

The following example assumes WebLogic Kubernetes Operator 4.1.2 is already installed.

Add the WebLogic helm chart catalog:
```text
ocne catalog add --uri https://oracle.github.io/weblogic-kubernetes-operator --name "WebLogic Kubernetes Operator"
```

Update the existing installation:
```text
ocne application update --release weblogic-operator --namespace verrazzano-system --version 4.1.2 --catalog "WebLogic Kubernetes Operator" --reset-values
```

