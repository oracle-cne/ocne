OCNE-NODE 1 "FEBRUARY 2024" Linux "User Manuals"
================================================

NAME
----

ocne node - Manage nodes inside a Kubernetes cluster

SYNOPSIS
--------

`ocne` `node` [`--kubeconfig` *path*] *subcommand*

DESCRIPTION
-----------

`ocne` `node` manages the lifecycle of individual nodes in a Kubernetes
cluster.

OPTIONS
-------

`-k`, `--kubeconfig` *path*
  A Kubernetes client configuration file that describes the target cluster as
  well as how to access it.  If this option is specified, all operations that
  work against an existing Kubernetes cluster will use this cluster.  This
  option takes precedence over the `KUBECONFIG` environment variable describe
  later in this document.

ENVIRONMENT
-----------

`KUBECONFIG`
  Behaves the same way as the `--kubeconfig` option.

SUBCOMMANDS
-----------

`update` [OPTIONS]...
  Updates the Kubernetes version of the target node, if an update is available.  The update is performed
  by cordoning the node, draining the node, staging the update, rebooting the
  node, and finally uncordoning the node once it has booted up to the new
  version.

`-N`, `--node` *name*
    The name of the node to update, as seen from within Kubernetes.  That is,
    the name should be one of the nodes listed in `kubectl` `get` `nodes`. [required]

`-t`, `--timeout` *string*
    Node drain timeout, such as 5m

`-d`, `--delete-emptydir-data` 
    Delete pods that use emptyDir during node drain. The default value is false

`-c`, `--disable-eviction`
    Force pods to be deleted during drain, bypassing PodDisruptionBudget. The default value is false.

SEE ALSO
--------

ocne-cluster(1)

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>
