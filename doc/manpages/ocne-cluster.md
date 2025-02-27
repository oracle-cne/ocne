OCNE-CLUSTER "FEBRUARY 2024" Linux "User Manuals"
=================================================

NAME
----

ocne cluster - Interact with Kubernetes clusters.

SYNOPSIS
--------

`ocne` `cluster` *subcommand*

DESCRIPTION
-----------

`ocne` `cluster` is a set of subcommands that deal with all aspects of cluster
management within Oracle Cloud Native Environment.  Please refer to individual
subcommands for details of what this set of commands can do.

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

`start` [OPTIONS]...
  Deploy a cluster from a given configuration.  There are four primary flavors
  of deployments: local virtualization, installation on to pre-provisioned
  compute resources, installation on to self-provisioned compute resources,
  and those that leverage a cloud provider or other infrastructure automation.

`-c`, `--config` *URI*
    The path to a configuration file that contains the definition of the
    cluster to create.  If this value is not provided, a small cluster is
    created using the default hypervisor for the system where the command
    is executed.

`-n`, `--control-plane-nodes` *integer*
    The number of control plane nodes to provision for clusters deployed using
    local virtualization.

`-w`, `--worker-nodes` *integer*
    The number of worker nodes to provision for clusters deployed using
    local virtualization.

`-u`,`--auto-start-ui` *bool*
    Determines if the web console is started automatically after the cluster
    has started.

`-C`,`--cluster-name` *string*
    The name of the cluster to start.  Naming clusters allows for the management
    of several clusters from the same system.

`-s`, `--session` *URI*
    Sets the session URI for the libvirt provider.

`-i`, `--key` *string*
    The ssh public key of the remote system. The default value is ~/.ssh/id_rsa.pub

`-o`, `--boot-volume-container-image` *URI*
    The URI of a container image that contains the OCK boot volume.

`-P`, `--provider` *string*
    The provider to use when interacting with the cluster.

`--virtual-ip` *string*
    The virtual IP address to use as the IP address of the Kubernetes API server.

`--load-balancer` *string*
    The hostname or IP address of the external load balancer for the Kubernetes API server.

`-v`,`--version` *version*
    The version of Kubernetes. This is the major and minor Kubernetes version.
    For example: 1.28
    If the version is not provided, then it defaults to the latest version known to the CLI.
    This version is ignored if the boot image is overridden.

`template` [OPTIONS]...
  Emits a sample cluster configuration that can be customized as needed.

`-P`, `--provider` *string*
    The provider to use when interacting with the cluster.

`-c`, `--config` *URI*
    The path to a configuration file that contains the definition of the
    cluster to create.  If this value is not provided, a small cluster is
    created using the default hypervisor for the system where the command
    is executed.

`delete` [OPTIONS]...
  Destroy a cluster that has been deployed using `ocne` `cluster` `start`.
  This command only applies to clusters that have been created using local
  virtualization.  It cannot be used to destroy clusters deployed to bare
  metal systems, pre-provisioned compute, or deployed using infrastructure
  automation APIs.

`-c`, `--config` *URI*
    The path to a configuration file that contains the definition of the
    cluster to delete.  If this value is not provided, it will destroy
    the small cluster that was generated by `ocne` `start` when run
    with no configuration file.

`-C`, `--cluster-name` *string*
    The name of the cluster to delete

`-s`, `--session` *URI*
    Sets the session URI for the libvirt provider.

`-P`, `--provider` *string*
    The provider to use when interacting with the cluster.

`dump` [OPTIONS]...
  Dump cluster resources and node data into a local directory. By default,
  a curated set of cluster resources from all namespaces and nodes are included. You 
  can select specific namespaces and nodes. Instead of curated cluster resources,
  you can include all cluster resources, except Secrets. Furthermore, Pod logs can
  also be included, but they are omitted by default.

`-c, --curated-resources`
    Dump manifests from a curated subset of cluster resources.  By default, 
    all cluster resources are dumped, except Secrets and ConfigMaps.

  `--json`
    Dump Kubernetes resources in JSON format. The default is false and dumps them in YAML format.

  `--managed`
    Include the managedFields data in the Kubernetes resources. The default is false.

`-m, --include-configmaps`
    Include ConfigMaps in the cluster dump. The --all-resources flag must also be true. 
    The default is false.

`-n, --namespaces` *string*
    A comma separated list of namespaces that should be dumped (e.g.,-n ns1,ns2).
    The default is all namespaces.

`-N, --nodes` *string*
    A comma separated list of nodes that should be dumped (e.g., -N node1,node2).
    The default is all nodes.

`-d, --output-directory` *string*
    The output directory where the dump files will be written.
    This is required.

`-c, --skip-cluster` 
    Skip dumping cluster resources. The default is false.

`-s, --skip-nodes`
    Skip dumping node resources. The default is false.

`-p, --skip-podlogs`
    Skip pod logs in the cluster dump. The default is false.

`-s, --skip-redaction`
    Skip the redaction of sensitive data.

`-z, --generate-archive` *string*
    The file path for an archive file to be generated by the command, instead of an output directory.


`info` [OPTIONS]...
  Get cluster information at the cluster level and from all the nodes in the cluster.
  You can select specific nodes or skip nodes altogether.

`-N, --nodes` *string*
    A comma separated list of nodes where information should be retrieved (e.g., -N node1,node2).
    The default is all nodes.

`-s, --skip-nodes`
    Skip dumping node resources. The default is false.


`join` [OPTIONS]...
  Join a node to a cluster, or generate the materials required to do so.
  This subcommand targets three cases: local virtualization, pre-provisioned
  compute, and self-provisioned compute.  For clusters created on local
  virtualization, new virtual machines are created and joined to the
  target cluster.  For pre-provisioned cases, it migrates a node from one
  cluster to another cluster.  For self-provisioned cases, it generates the
  materials needed to join a node to a cluster on first boot.
  
  At least one of `--control-plane-nodes` or `--worker-nodes` must be
  provided for the local virtualization and self-provisioned cases.

`-n`, `--control-plane-nodes` *integer*
    The number of control plane nodes to provision for clusters deployed using
    local virtualization.

`-c`, `--config` *URI*
    The path to a configuration file.

`-w`, `--worker-nodes` *integer*
    The number of worker nodes to provision for clusters deployed using
    local virtualization.

`-d`, `--destination` *path*
    The path to a Kubernetes client configuration that describes the cluster
    that the node will join.

`-N`, `--node` *name*
    The name of the node to move from the source cluster to the destination
    cluster, as seen from within Kubernetes.  That is, the name should be one
    of the nodes listed in `kubectl` `get` `nodes`.

`-r`, `--role-control-plane`
    When moving a node from one cluster to another, specify this flag to join
    the node as a control plane. The default is to join the node as a worker.

`-P`, `--provider` *string*
    The provider to use when interacting with the cluster.

`backup` [OPTIONS]...
  Backup the contents of the `etcd` database that stores data for a target
  cluster.

`-o`, `--out` *path*
    The location where the backup materials are written. [required]

`analyze`
  Analyze a live cluster or a cluster dump.  If no dump directory or archive file is specified, 
  then the live cluster will be analyzed.  Analyze will report problems that it discovers.

  `-d, --output-directory` *string*
  The output directory containing cluster dump data which will be analyzed.

`  -s, --skip-nodes`
  Skip collecting node resources for the analysis, default is false. This is only valid for a live cluster analysis.

  `-p, --skip-pod-logs`
  Skip collecting pod logs for the analysis, default is false. This is only valid for a live cluster analysis.

  `-v, --verbose`
  Display additional detailed information related to the analysis.

`console` [OPTIONS]...
  Launch an administration console on nodes in a Kubernetes cluster. Use the --direct option
  to access the local filesystem of the target node.

`-N`, `--node` *node-name*
    The Kubernetes cluster node where the console is to be launched. [required]

`-t`, `--toolbox`
    Create the console using a container image that contains a variety of tools
    that are useful for diagnosing a Linux system.

`list`
  Lists all known clusters

`show` [OPTIONS]...
  Shows details for a cluster.

`-C`, `--cluster-name` *string*
    The name of the cluster to delete

`-a`, `--all`
    Displays the complete configuration for the cluster

`-f`, `--field` *field*
    The YAML path to a section of the configuration to show

`stage` [OPTIONS]...
  Sets the kubernetes version of all nodes and updates the Kubernetes version of the cluster.
  Staging an update prompts each node to download the requested update.
  Once the update is available, each node update must be manually applied.

`-v`, `--version` *string*
    The version of Kubernetes to update to. [required]

`-t`, `--transport` *string*
    The type of transport to use during an upgrade.

`-r`, `--os-registry` *URI*
    The name of the os registry to use during an upgrade.

SEE ALSO
--------

ocne-config.yaml(5) ocne-node(1)

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>
