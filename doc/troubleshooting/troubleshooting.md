# Troubleshooting Kubernetes Clusters

Kubernetes clusters are complex ecosystems that interact with all parts of
the software and infrastructure stack.  Any number of things can and do go
wrong.  Due to the number of things that can go wrong, it is often difficult
to know where to start looking to diagnose a problem.

## Cluster Dumps

Taking a cluster dump allows for manual inspection of cluster state, and makes
it easy to transmit or store state for future analyis.  The contents and
structure of a dump is not a stable interface and is subject to change at
all times.

### Generating Dumps

Dumps can be stored in a directory structure to allow immediate manual analysis.
```
$ ocne cluster dump --output-directory ./dump
INFO[2024-08-02T19:54:24Z] Collecting node data                         
INFO[2024-08-02T19:54:24Z] Collecting cluster data                      
INFO[2024-08-02T19:54:36Z] Cluster dump successfully completed, files written to /home/opc/tmp/ocne/dump 

$ ls dump
cluster  cluster-info.out  nodes
```

An archive can be created for easy transport.
```
$ ocne cluster dump --generate-archive dump.tgz
INFO[2024-08-02T19:53:41Z] Collecting node data                         
INFO[2024-08-02T19:53:41Z] Collecting cluster data                      
INFO[2024-08-02T19:53:54Z] Cluster dump successfully completed, archive file written to dump.tgz
```

### Dump Contents

A cluster dump has a variety of data, sorted in directories.
```
$ ls ./dump/nodes/REDACTED-6b341247/
crictl.yaml  disk-du.out        journal.crio.service    journal.errors.out  journal.kubelet.service     journal.ocne.service         journal.ostree-remount.service  ostree-refs.out  top.out
disk-df.out  journal.boots.out  journal.disk-usage.out  journal.head.out    journal.ocne-nginx.service  journal.ocne-update.service  journal.tail.out                services.out     update.yaml

$ ls dump/cluster/namespaces/
default  kube-flannel  kube-node-lease  kube-public  kube-system  ocne-system

$ ls dump/cluster/namespaces/kube-flannel/
controllerrevisions.apps.json  daemonsets.apps.json  events.events.k8s.io.json  events.json  podlogs  pods.json  serviceaccounts.json
```

## Cluster Analysis

Cluster analysis can be perfomed against a dump archive or a running cluster.
The analysis tool inspects the contents of a dump and reports surprising state
and potential solutions to common problems.  Note that this is a debug utility
meant to inform the diagnosis of a problem.  Results are not guaranteed.

Analyzing a healthy cluster will report no issues.
```
ocne cluster analyze
INFO[2024-08-02T19:59:18Z] Collecting node data

Cluster Nodes:
--------------
Cluster nodes are normal
```

If a node is unschedulable, due to a cordoning for example, it will be reported.
```
$ kubectl get nodes
NAME        STATUS   ROLES           AGE     VERSION
ocne19429   Ready    <none>          6m41s   v1.27.12+1.el8
ocne23631   Ready    <none>          6m40s   v1.27.12+1.el8
ocne3548    Ready    control-plane   7m16s   v1.27.12+1.el8

$ kubectl cordon ocne3548
node/ocne3548 cordoned

$ ocne cluster analyze
INFO[2024-08-02T20:03:01Z] Collecting node data

Cluster Nodes:
--------------
Pods cannot be scheduled on node ocne3548

```

A dump can be analyzed.
```
$ ocne cluster dump --generate-archive dump.tgz
INFO[2024-08-02T20:12:06Z] Collecting node data                         
INFO[2024-08-02T20:12:06Z] Collecting cluster data                      
INFO[2024-08-02T20:12:17Z] Cluster dump successfully completed, archive file written to dump.tgz 

$ ocne cluster analyze --dump-directory ./dump

Cluster Nodes:
--------------
Pods cannot be scheduled on node REDACTED-92bb1a8e

```
