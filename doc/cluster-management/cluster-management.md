# Cluster Management

Kubernetes clusters are managed using the `ocne cluster` command and its
subcommands.  Clusters can be created and destroyed.

## Providers

Clusters can be started on multiple platforms.  See the provider specific
pages for details on how to use each one.

* [Libvirt](libvirt.md)
* [OCI](oci.md)
* [BYO](byo.md)

## Cluster Lifecyle

Kubernetes clusters have a complex lifecycle.  They can be created, destroyed,
upgrade, grown, shrunk, among other things.  The commands listed here are
representative of the lifecycle of a cluster, but are not useful on their own.
Please refer to the documentation for a specific provider to see how a cluster
is actually managed in that environment.

Cluster are created with:
```
$ ocne cluster start
```

Clusters are destroyed with:
```
$ ocne cluster delete
```

Clusters are updated with:
```
$ ocne cluster stage
```

Nodes in clusters are updated with:
```
$ ocne node update
```

## Viewing Clusters

Clusters that have been created are maintained in a local inventory.  The set of
clusters can be listed.

```
$ ocne cluster list
ocne
oci-ci
```

Detailed settings can be viewed.  By default, the kubeconfig for the cluster is
shown.  It is also possible to extract specific fieds, or show the full
configuration.

Show the kubeconfig
```
$ ocne cluster show -C ocne 
kubeconfig: /home/myuser/.kube/kubeconfig.ocne
```

Show the complete configuration
```
# ocne cluster show -C ocne -a
[opc@instance-20240110-1300 ocne]$ ./out/linux_amd64/ocne cluster show -C ocne -a
config:
    name: ocne
    provider: libvirt
    providers:
        libvirt:
            uri: qemu:///session
            sshKey: ""
...
    kubernetesVersion: 1.28.3
    sshPublicKeyPath: ~/.ssh/id_rsa.pub
    sshPublicKey: ssh-rsa mysshpublickey
    clusterDefinitionInline: ""
    clusterDefinition: ""
```

Extract specific fields
```
$ ocne cluster show -C ocne -f "config.providers.libvirt"
uri: qemu:///session
sshKey: ""
storagePool: images
network: default
controlPlaneNode:
    memory: 4194304Ki
    cpu: 2
    storage: 15Gi
workerNode:
    memory: 4194304Ki
    cpu: 2
    storage: 15Gi
bootVolumeName: boot.qcow2
bootVolumeContainerImagePath: disk/boot.qcow2
```
