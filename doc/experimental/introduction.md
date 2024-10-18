# Introduction

### Version: v0.0.2-draft

## Migration Summary

Migration is divided into three main phases:
1. Verrazzano migration
2. Oracle Cloud Native Environment 2.0 OCK in-place upgrade
3. Migration from unsupported components.

### Phase One: Verrazzano Migration to Catalog
Verrazzano migration means that Verrazzano is no longer used and all application lifecycle management of supported
components (applications) can be done via the Oracle Cloud Native Environment CLI using the catalog.

When this phase is complete, Verrazzano is effectively removed from the system and OAM resources no longer exist.
Nothing in the topology or system architecture changes during this phase. All the components initially installed
and configured by Verrazzano will continue to work, except for certain obsolete components that are no longer required.

The **acceptance criteria** for this phase being done follows:

1. Oracle Cloud Native Environment 2.0 CLI is used for application lifecycle management
2. Oracle Cloud Native Environment 2.0 catalog and UI installed on the cluster
3. The lifecycle of all supported components can be done via the Oracle Cloud Native environment 2.0 catalog
4. The installed versions of all components can be used, they do not need to be upgraded
5. OAM resources translated to Kubernetes native resource YAML 
6. Verrazzano controllers and related resources are removed from cluster
7. Some Oracle Cloud Native Environment 2.0 CLI functionality is not yet available (stage, update, etc.)

### Phase Two: Oracle Cloud Native Environment 2.0 OCK in-place Upgrade
Oracle Cloud Native Environment 2.0 OCK migration means that all the Kubernetes hosts are running OCK images with Oracle Cloud Native Environment 2.0, instead of 1.x. 
This phase will require in-place migration where no new nodes are added, rather existing nodes are updated to use the OCK 2.0 image.

The **acceptance criteria** for this phase being done follows:

1. All nodes in the cluster are running OCK 2.0 images
2. All Oracle Cloud Native Environment 2.0 CLI functionality can be used
2. Kubernetes version upgrade is not required if the existing 1.x system has Kubernetes 1.26-1.30

###  Phase Three: Unsupported Components that Require Migration
There are a few obsolete or unsupported components that are currently integrated with supported components.
For example, Fluentd and Grafana require auth-proxy.  This phase discusses those solutions.

The **acceptance criteria** for this phase being done follows:

1. The auth-proxy can be removed from system without breaking supported consoles, logging, and identity.
2. Certain unsupported components on the system have Helm charts and are deployed as Helm release, so they can be upgraded.

## Obsolete and Unsupported Components
This section lists components that are either obsolete or do not require migration.

### Obsolete components 
These components are obsolete and were removed from the system during migration.

* Verrazzano Platform Operator
* Verrazzano Application Operator
* Verrazzano Monitoring Operator
* Verrazzano Cluster Operator
* Verrazzano CAPI controllers

### Unsupported components
These components are unsupported, but not removed.
You may continue to use these components at your own discretion.

* Argo CD
* Coherence
* Rancher
* Thanos

---
[Next: Phase One](./phase1/phase1.md)