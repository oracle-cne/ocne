# Phase Two: Oracle Cloud Native Environment 2.0 OCK Migration

### Version: v0.0.5-draft

## Overview
Instructions for performing an in-place upgrade of a Kubernetes cluster from Oracle Cloud Native Environment 1.x to 2.x.

## Prerequisites
Identify the VM type and Kubernetes of the existing Oracle Cloud Native Environment 1.x cluster and generate the OS image.

### Shutdown the Module Operator
Scale-in the verrazzano-module-operator so that it does not process any Module CRs.
```text
kubectl scale deployment verrazzano-module-operator -n verrazzano-module-operator  --replicas=0
```

## Upgrade all nodes to OCK 2.*

Set up a byo cluster for the existing Oracle Cloud Native Environment 1.x cluster using the 2.0 CLI. 
The example below assumes Kubernetes 1.26, change it to your version.

1. Create an OCI OCK image
    ```
    ocne image create --arch amd64 --type oci --version 1.26
    ocne image upload --arch amd64 --type oci --version 1.26 --bucket <oci-bucket-name> --compartment <oci-compartment-name> --image-name ocnos126 --file /home/opc/.ocne/images/boot.qcow2-1.26-amd64.oci
    ```
### Upgrade all the nodes
Starting with the control plane nodes, upgrade all the cluster nodes to OCK 2.*.

See [Upgrade Single Node to OCK 2.*](../phase2/ock-upgrade.md)

## Verify that all the nodes are runnnig OCK 2.*

To confirm that the entire cluster is using OCK 2.*, run the following:
```text
ocne cluster info --kubeconfig $KUBECONFIG 
```

Make sure the following information is displayed for each cluster node.  If it isn't, then
go back and upgrade the node to use OCK.  
```text
...

Node: ocne-control-plane-1
  Registry and tag for ostree patch images:
    registry: container-registry.oracle.com/olcne/ock-ostree
    tag: 1.26
    transport: ostree-unverified-registry
  Ostree deployments:
      ock 5d6e86d05fa0b9390c748a0a19288ca32bwer1eac42fef1c048050ce03ffb5ff9.1 (staged)
    * ock 5d6e86d05fa0b9390c748a0a19288ca32bwer1eac42fef1c048050ce03ffb5ff9.0
    
Node: ocne-control-plane-2
  Registry and tag for ostree patch images:
    registry: container-registry.oracle.com/olcne/ock-ostree
    tag: 1.26
    transport: ostree-unverified-registry
  Ostree deployments:
      ock 5d6e86d05fa0b9390c748a0a19288ca32bwer1eac42fef1c048050ce03ffb5ff9.1 (staged)
    * ock 5d6e86d05fa0b9390c748a0a19288ca32bwer1eac42fef1c048050ce03ffb5ff9.0
    
etc.      
```

If the node is NOT using OCK then you will be missing the Registry and Ostree details like the following:
```text
Node: ocne-control-plane-1
  Registry and tag for ostree patch images:
  Ostree deployments:
```

## Uninstall the Module Operator and Delete Module CRs
After the entire has been upgraded to OCK, then you can safely remove the verrazzano-module-operator and Module CRs.

```text
 helm uninstall -n verrazzano-module-operator verrazzano-module-operator --wait
```

### Delete the Module CRs
There are several CRs of type Module. Once all the nodes have moved to OCK, then they are no longer needed.  
They can be found by the following kubectl command 
```text
kubectl get Module -A
```
At this point, the module operator should be removed from the system, so the finalizers have to be deleted. 
We have instructions for other resources with similar requirements. See https://github.com/oracle-cne/ocne/blob/main/doc/experimental/phase1/oam-remove-objects.md

## Delete all the Verrazzano related CRDs
```text
kubectl delete crd ingresstraits.oam.verrazzano.io
kubectl delete crd loggingtraits.oam.verrazzano.io
kubectl delete crd metricsbindings.app.verrazzano.io
kubectl delete crd metricstemplates.app.verrazzano.io
kubectl delete crd metricstraits.oam.verrazzano.io
kubectl delete crd modules.platform.verrazzano.io  
kubectl delete crd multiclusterapplicationconfigurations.clusters.verrazzano.io
kubectl delete crd multiclustercomponents.clusters.verrazzano.io
kubectl delete crd multiclusterconfigmaps.clusters.verrazzano.io
kubectl delete crd multiclustersecrets.clusters.verrazzano.io 
kubectl delete crd verrazzanocoherenceworkloads.oam.verrazzano.io
kubectl delete crd verrazzanohelidonworkloads.oam.verrazzano.io
kubectl delete crd verrazzanomonitoringinstances.verrazzano.io
kubectl delete crd verrazzanoprojects.clusters.verrazzano.io
kubectl delete crd verrazzanos.install.verrazzano.io
kubectl delete crd verrazzanoweblogicworkloads.oam.verrazzano.io
```

---
[Next: Phase Three](../phase3/phase3.md)  
[Previous: Phase One](../phase1/phase1.md)