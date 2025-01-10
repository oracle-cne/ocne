OCNE-CONFIG.YAML 5 "FEBRUARY 2024" Linux "User Manuals"
=======================================================

NAME
----

ocne-config.yaml - Cluster configuration for Oracle Cloud Native Environment

DESCRIPTION
-----------

This file contains the configuration for a Kubernetes cluster deployed by the
Oracle Cloud Native Environment.  A configuration file is meant to be used as
a template to recreate Kubernetes clusters in various environments without having
to maintain and synchronize multiple identical files for each cluster.

SCHEMA
------

```
# The name of the cluster
name: ocne

# The number of control plane nodes to spawn
# When the provider is set to oci, the number of control plane nodes must be odd
controlPlaneNodes: 1

# The number of worker nodes to spawn
workerNodes: 0

# When using the built-in keepalived/nginx load balancer
# the cluster is configured to use this IP.  In "libvirt" provider,
# if neither loadBalancer nor virtualIp is provided,
# a virtual IP is generated from the subnet of the control plane nodes.
virtualIp: 192.168.0.100

# The hostname or IP address of the external load balancer
# for the Kubernetes API server.  Do not use it if the virtualIp
# option is used.
loadBalancer: 100.100.1.1

# The provider used to deploy the cluster.  Valid values are "libvirt"
# and "VirtualBox".
provider: libvirt

# Any proxy parameters that need to be propagated into the cluster nodes.
# These values are set for any on-host services that require a network
# connection.
proxy:
  httpsProxy: http://myproxy:2138
  httpProxy: http://myproxy:2138
  noProxy: somehost.localdomain

# The Container Networking Interface (CNI) provider to install when the cluster
# is instantiated.  The value can be any CNI offered with Oracle Cloud Native
# Environment, or "none" if another CNI will be deployed either manually or via
# an application catalog.
#
# Note: Multus cannot be used as a primary CNI.
cni: flannel

# The subnet to use for the service network.
serviceSubnet: 10.96.0.0/12

# The mode for kube-proxy.  Can be one of either "iptables" or "ipvs"
kubeProxyMode: iptables

# The subnet to use for the pod network.  The chosen CNI is automatically
# configured to use this subnet.
podSubnet: 10.244.0.0/16

# Determines if the Oracle Cloud Native UI is installed.  If the value is set
# to true, then the UI is installed.  If if the value is false, then it will
# not.  The default value is "true".
headless: false

# Determines of "ocne cluster start" automatically opens a tunnel to
# the UI service and opens a browser with the UI.
autoStartUI: true

# Reduces the number of messages printed by "ocne"
quiet: false

# All cluster nodes are provisioned with this container image
# registry as the default registry to search for partially
# qualified container images.
registry: container-registry.oracle.com

# Specifies the port that the Kubernetes API is exposed on
kubeApiServerBindPort: 6443

# Specifies the port that kube-apiserver is actually listening on
# when deploying with the keepalived/nginx based self-hosted
# load balancer.
kubeApiServerBindPortAlt: 6444

# Specifies the cipher suites used by Kubernetes cluster, If omitted,
# the default Go cipher suites will be used.
cipherSuites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384

# For clusters that use complex configuration that is not
# provided by this configuration file, this is the path
# to the file with the extra configuration.
# Note: This field is not applicable to libvirt and byo providers.
clusterDefinition: mycluster.yaml

# For clusters that use complex configuration that is not
# provided by the configuration file, this value can be
# used to specify in-line configuration.  This option
# cannot be used with "clusterDefinition".
# Note: This field is not applicable to libvirt and byo providers.
clusterDefinitionInline: |
  somekey: someval
  someotherkey: someotherval
  
# This field specifies the password set for the "ocne" user.
# This configuration is applied through ignition. Certain providers
# require an ignition file to be passed in with the desired password
# specified to enable log in. The "ocne" user is useful for obtaining the 
# kubeconfig of a successful OCK instance and to access the rescue shell.
# The rescue shell is only available when Oracle Cloud Native Environment fails to start
# properly on that instance. The password hash must be generated
# using SHA512. 
password: <some-password-hash>

# A list of catalogs to deploy to the target cluster after it has been
# instantiated.
catalogs:
  # The "name" field indicates the name of the catalog.  It is used
  # by other Oracle Cloud Native Environment interfaces to refer to
  # the catalog.
  #
  # The "uri" field indicates the source of the catalog.  Depending
  # on the source, a pod may or may not be deployed into the
  # cluster that serves the catalog content.
  - name: mycatalog
    uri: https://mycatalog.mydomain/catalog


# A list of modules to deploy to the target cluster after it has been
# instantiated.
applications:
  # The "application" field is the name of the application to install
  # from a catalog.
  #
  # The "name" field is the name of this deployment of the application.  It
  # is useful for applications that may benefit from having multiple
  # installations within a single cluster.
  #
  # The "catalog" field indicates which catalog contains the application.
  #
  # The "config" field contains a URI that points to a configuration file
  # for the application.
  - name: multus
    release: mymultus
    catalog: oracle
    config: git://gitlab.mydomain/configs/multus.yaml

# Provider-specific configuration options.
providers:
  libvirt:
    # Default value for the libvirt connection URI
    uri: qemu:///system
    # SSH keyfile to use for ssh-based connections
    sshKey: /home/myuser/.ssh/id_rsa.ocne
    # The storage pool to use for images
    storagePool: mypool
    # The virtual network to use for domains
    network: bridge-1
    # Boot volume name
    bootVolumeName: boot.qcow2
    # Boot volume container image path
    bootVolumeContainerImagePath: disk/boot.qcow2
    # Configuration options for control plane and worker nodes.
    # For values that have units of bytes, suffixes like M
    # or G are in megabytes and gigabytes while suffixes like
    # Mi or Gi are in mebigytes and gibibytes.
    controlPlaneNode:
      # Number of CPUs
      cpu: 2
      # Amount of memory
      memory: 16Gi
      # Size of disk
      storage: 8Gi
    workerNode:
      cpu: 2
      memory: 16Gi
      storage: 8Gi
  oci:
    # The kubeconfig file for the target management cluster
    kubeconfig: /home/myuser/.kube/kubeconfig.mgmt
    # The compartment to deploy OCI resources in to.  It can
    # be either the path to a compartment (e.g. mytenancy/mycompartment)
    # or the OCID of a compartment.
    compartment:
    # The Kubernetes namespace where Cluster API resources
    # should be deploye.d
    namespace:
    # The OCIDs of the OCIs compute image to use as the initial
    # disk image for any compute resources.
    images:
      amd64:
      arm64:
    # The shape of the compute instance for control plane nodes
    controlPlaneShape:
      shape: VM.Standard.A1.Flex
      ocpus: 2
    # The shape of compute instances for worker nodes
    workerShape:
      shape: VM.Standard.E4.Flex
      ocpus: 4
    # Indicates if a cluster is self-managing or not.  If set to
    # true, the cluster will contain all the necessary controllers
    # and resources to manage its own lifecycle.  If not, those
    # resources will remain in the initial admin cluster.
    selfManaged: false
    # The subnets to use when provisioning OCI load balancers for
    # default deployments of OCI-CCM
    loadBalancer:
      subnet1:
      subnet2:
    # The OCID of the VCN to use when creating load balancers for
    # default deployments of OCI-CCM
    vcn:
    # The bucket where OCK boot images are stored when they are uploaded
    # to OCI object storage.
    imageBucket: ocne-images
  byo:
    # If set to true, any time that a join token is required it is created
    # automatically as part of the command.  If it is false, the token must
    # be created manually.
    automaticTokenCreation: false
    # Specify the network interface that the CNI and other Kubernetes services
    # should bind to.  This value is required.
    networkInterface:

# Allows customization of any short-lived clusters that may be spawned
# to perform tasks that cannot be accomplished on the host system.  It
# often used for things like modifying boot images or deploying Cluster API
# resources.
ephemeralCluster:
  # The name of the cluster
  name: ocne-ephemeral
  # Indicates if the cluster should be automatically
  # deleted after the work is complete.
  preserve: false
  # VM level configuration
  node:
    # The number of CPUs to assign to the VM
    cpus: 2
    # The amount of RAM the VM has
    memory: 4GB
    # The size of the root disk for the VM
    storage: 15GB

# The container image registry and tag that contains a
# bootable OCK image
bootVolumeContainerImage: container-registry.oracle.com/olcne/ock:1.30

# The kubeconfig to use for operations that require a running cluster
kubeconfig: /home/myuser/.kube/kubeconfig.utilitycluster
```

SEE ALSO
--------

ocne-cluster(1) ocne-catalog(1) ocne-application(1)

AUTHOR
------

Daniel Krasinski <daniel.krasinski@oracle.com>
