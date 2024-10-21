# Remove OAM objects

### Version: v0.0.1-draft
This document explains how to remove the OAM objects.

***WARNING*** Before following these steps, you MUST complete the steps to remove Verrazzano controllers from the system, 
see [instructions](../phase1/disable-verrazzano.md).  If you proceed without doing that your application WILL get deleted.

The deletion happens in two steps: First, remove the finalizers then delete the objects.

## Remove the finalizers

### List the OAM objects
First get all the OAM objects on the system
```text
 kubectl get --all-namespaces applicationconfigurations
 kubectl get --all-namespaces components
 kubectl get --all-namespaces containerizedworkloads
 kubectl get --all-namespaces healthscopes
 kubectl get --all-namespaces manualscalertraits
 kubectl get --all-namespaces scopedefinitions
 kubectl get --all-namespaces traitdefinitions
 kubectl get --all-namespaces workloaddefinitions
 kubectl get --all-namespaces ingresstraits
 kubectl get --all-namespaces loggingtraits
 kubectl get --all-namespaces metricstraits
 kubectl get --all-namespaces verrazzanocoherenceworkloads
 kubectl get --all-namespaces verrazzanohelidonworkloads
 kubectl get --all-namespaces verrazzanoweblogicworkloads
```

### Remove finalizer for each object
Remove the finalizers for each object.  For example, here are some of the todo-list objects.
```text
 kubectl patch  ingresstrait -n todo-list  todo-domain-ingress -p '{"metadata":{"finalizers":[]}}' --type=merge
 
 kubectl patch  metricstrait -n todo-list todo-domain-trait-7d455c67bc  -p '{"metadata":{"finalizers":[]}}' --type=merge
 kubectl patch  metricstrait -n todo-list todo-mysql-deployment-trait-656cfbdb97 -p '{"metadata":{"finalizers":[]}}' --type=merge
 
 etc.
```

## Delete OAM objects
```text
 kubectl delete --all --all-namespaces applicationconfigurations --cascade=orphan
 kubectl delete --all --all-namespaces components --cascade=orphan
 kubectl delete --all --all-namespaces containerizedworkloads --cascade=orphan
 kubectl delete --all --all-namespaces healthscopes --cascade=orphan
 kubectl delete --all --all-namespaces manualscalertraits --cascade=orphan
 kubectl delete --all --all-namespaces scopedefinitions --cascade=orphan
 kubectl delete --all --all-namespaces traitdefinitions --cascade=orphan
 kubectl delete --all --all-namespaces workloaddefinitions --cascade=orphan
 kubectl delete --all --all-namespaces ingresstraits --cascade=orphan
 kubectl delete --all --all-namespaces loggingtraits --cascade=orphan
 kubectl delete --all --all-namespaces metricstraits --cascade=orphan
 kubectl delete --all --all-namespaces verrazzanocoherenceworkloads --cascade=orphan
 kubectl delete --all --all-namespaces verrazzanohelidonworkloads --cascade=orphan
 kubectl delete --all --all-namespaces verrazzanoweblogicworkloads --cascade=orphan
```
## Confirm that the OAM objects are deleted
Each of the following commands should return `no resources found`.  If not, repeat the previous steps of removing the finalizers.

```text
 kubectl get --all-namespaces applicationconfigurations
 kubectl get --all-namespaces components
 kubectl get --all-namespaces containerizedworkloads
 kubectl get --all-namespaces healthscopes
 kubectl get --all-namespaces manualscalertraits
 kubectl get --all-namespaces scopedefinitions
 kubectl get --all-namespaces traitdefinitions
 kubectl get --all-namespaces workloaddefinitions
 kubectl get --all-namespaces ingresstraits
 kubectl get --all-namespaces loggingtraits
 kubectl get --all-namespaces metricstraits
 kubectl get --all-namespaces verrazzanocoherenceworkloads
 kubectl get --all-namespaces verrazzanohelidonworkloads
 kubectl get --all-namespaces verrazzanoweblogicworkloads
```


## Check application the start the WebLogic operator
Validate that the domain resource(s) exist and that the application is accessible.
