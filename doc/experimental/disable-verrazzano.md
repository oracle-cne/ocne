# Remove Obsolete and Unused Verrazzano Components

### Version: v0.0.1-draft

The purpose of this document is to remove Verrazzano controllers that are obsolete and not used.  

Once these operators are removed, the following custom resources will be ignored:

* metricsbindings.app.verrazzano.io 
* metricstemplates.app.verrazzano.io 
* verrazzanos.install.verrazzano.io
* components.core.oam.dev
* containerizedworkloads.core.oam.dev
* healthscopes.core.oam.dev
* ingresstraits.oam.verrazzano.io
* loggingtraits.oam.verrazzano.io
* manualscalertraits.core.oam.dev
* metricstraits.oam.verrazzano.io
* scopedefinitions.core.oam.dev
* traitdefinitions.core.oam.dev
* verrazzanocoherenceworkloads.oam.verrazzano.io
* verrazzanohelidonworkloads.oam.verrazzano.io
* verrazzanomanagedclusters.clusters.verrazzano.io
* verrazzanomonitoringinstances.verrazzano.io
* verrazzanoweblogicworkloads.oam.verrazzano.io

## Remove the Verrazzano Platform Operator

```text
# Scale the deployments to zero replicas
kubectl scale deployment verrazzano-platform-operator --namespace verrazzano-install --replicas 0
kubectl scale deployment verrazzano-platform-operator-webhook --namespace verrazzano-install --replicas 0

# Verify the deployments in verrazzano-install have zero ready pods
kubectl get deployment -n verrazzano-install
NAME                                   READY   UP-TO-DATE   AVAILABLE   AGE
verrazzano-platform-operator           0/0     0            0           34m
verrazzano-platform-operator-webhook   0/0     0            0           34m

# Delete the verrazzano-install namespace
kubectl delete namespace verrazzano-install

# Delete associated WebHook configurations
kubectl delete validatingwebhookconfiguration verrazzano-platform-operator-webhook
kubectl delete validatingwebhookconfiguration verrazzano-platform-requirements-validator
kubectl delete validatingwebhookconfiguration verrazzano-platform-mysqlinstalloverrides
```

## Remove Remaining Verrazzano Controllers

```text
# Remove operators deployed using Helm
helm delete -n verrazzano-system verrazzano-application-operator
helm delete -n verrazzano-system verrazzano-cluster-operator
helm delete -n verrazzano-system verrazzano-monitoring-operator
helm delete -n verrazzano-system oam-kubernetes-runtime

# Verify the deployments in verrazzano-system have zero ready pods
kubectl get deployment -n verrazzano-system | grep operator | grep verrazzano
verrazzano-application-operator           0/0     0            0           8d
verrazzano-application-operator-webhook   0/0     0            0           8d
verrazzano-cluster-operator               0/0     0            0           8d
verrazzano-cluster-operator-webhook       0/0     0            0           8d
verrazzano-monitoring-operator            0/0     0            0           8d

# Delete associated WebHook Configurations
kubectl delete mutatingwebhookconfiguration verrazzano-mysql-backup
```