# Phase One: Verrazzano Migration

### Version: v0.0.1-draft

The instructions must be performed in the sequence outlined in this document.

## Turn off the Verrazzano controllers

Follow these [instructions](./disable-verrazzano.md) to disable the Verrazzano controllers on the cluster.

## Install Oracle Cloud Native Environment 2.0 CLI

Follow these [instructions](https://docs.oracle.com/en/operating-systems/olcne/2.0/cli/ocne_install_task.html#ocne_install) to install the 2.0 CLI on the cluster.

## Install Oracle Cloud Native Environment 2.0 Catalog and UI

```text
ocne cluster start --provider none --kubeconfig $KUBECONFIG --auto-start-ui false
kubectl -n ocne-system rollout status deployment ocne-catalog
```