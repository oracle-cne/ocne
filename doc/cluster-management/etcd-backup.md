# Etcd Backup

The state for Kubernetes clusters is maintained in an etcd database.  Access
to the database is shared between all Kubernetes API Server instances.  Taking
backups of the etcd database is a critical portion of a Kubernetes disaster
recovery plan.

Etcd backups are not a complete solution.  Etcd only contains the definitions
for Kubernetes resource definitions.  It does not contain any application data
or the contents of any volumes mounted within application pods.  Other
utilities are required for a complete disaster recovery solution.

Etcd backups are not a rollback mechanism.  They contain a snapshot of the state
of Kubernetes cluster resources at a moment in time.  Any changes to application
configuration or state will not be reflected if an etcd cluster is restored
from a backup.  The backup may contain Kubernetes resources that reference
old versions of Kubernetes cluster components or are not even supported by the
current Kubernetes cluster version.

## Taking a Backup

Etcd backups are taken with the `ocne cluster backup` subcommand:
```
$ ocne cluster backup --out etcd-snapshot.db
INFO[2024-07-03T16:11:19Z] Waiting for pod kube-system/etcd-ocne17251 to be ready: ok 
INFO[2024-07-03T16:11:19Z] Running etcd backup on pod etcd-ocne17251    
INFO[2024-07-03T16:11:20Z] Copying data from etcd pod to the local system: ok 
INFO[2024-07-03T16:11:20Z] Etcd successfully backed up


$ ls -l etcd-snapshot.db
-rw-rw-r--. 1 opc opc 2183200 Jul  3 16:11 etcd-snapshot.db
```

## Restoring From a Backup

Refer to the [Kubernetes etcd restore](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/#restoring-an-etcd-cluster) documentation for instructions on restoring etcd clusters.
Note that cluster services will not be available during the restore process.
Applications may or may not be affected during the restore process or after
it has completed.

## Considerations

Etcd backups contain the complete set of Kubernetes resources in a cluster at
the time the backup is taken.  Typically this will include sensitive data such
as Kubernetes Secret objects.  Care should be taken to put these backups in
a secure location.

If restoring from an etcd backup is part of a disaster recovery strategy, the
integrity of the backup file is important.  Backups should be stored in a
location with sufficient integrity guarantees.
