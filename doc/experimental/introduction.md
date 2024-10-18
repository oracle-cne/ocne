# Introduction

### Version: v0.0.3-draft

## Migration Summary

Migration is divided into three main phases:
1. Verrazzano migration.
2. Oracle Cloud Native Environment 2.0 OCK in-place upgrade.
3. Migration from auth-proxy.

These phases must be done in order. The phases describe discreet segments of the migration where the system remains
stable and the end of each phase, providing a methodical migration with no disruption of service.  For example, 
once phase one has been completed, then the system is ready for phase two. However, there can be a delay between 
phase one and phase two as required, which might be days or even weeks.

Each phase has acceptance criteria that explicitly describes what is true at the end of the phase.

### Phase One: Verrazzano Migration to Catalog
Verrazzano migration means that Verrazzano is no longer used and all application lifecycle management of supported
components (applications) can be done via the Oracle Cloud Native Environment CLI using the catalog.

When this phase is complete, Verrazzano is effectively removed from the system and OAM resources no longer exist.
Nothing in the topology or system architecture changes during this phase. All the components initially installed
and configured by Verrazzano will continue to work, except for certain obsolete components that are no longer required.

***NOTE***
Verrazzano CAPI is not supported in Oracle Cloud Native Environment 2.0. If you have an Oracle Cloud Native Environment 1.7 CAPI
cluster then you will need to recreated it with Oracle Cloud Native Environment 2.0 CLI.

The **acceptance criteria** for this phase being done follows:
 
1. Verrazzano controllers and related resources, including OAM, are removed from the cluster.
2. All installed applications will continue to work.
3. All consoles and ingresses will continue to work.
4. OAuth2 security provided by auth-proxy and Keycloak stack (Keycloak, MySQL, MySQL Operator) will continue to work.
5. Installed components will continue to work, but some need to be upgraded to a newer version as described in the phase one document.
6. The Oracle Cloud Native Environment 2.0 catalog and UI are installed on the cluster.
7. The lifecycle of all supported components can be done with the Oracle Cloud Native Environment 2.0 CLI using the catalog.
8. Some Oracle Cloud Native Environment 2.0 CLI functionality is not yet available (stage, update).
9. Existing Verrazzano CAPI is not supported, it is replaced by Oracle Cloud Native Environment 2.0 CAPI.

### Phase Two: Oracle Cloud Native Environment 2.0 OCK in-place Upgrade
Oracle Cloud Native Environment 2.0 OCK migration means that all the Kubernetes hosts are running OCK images with Oracle Cloud Native Environment 2.0, instead of 1.x. 
This phase will require in-place migration where no new nodes are added, instead, existing nodes are updated to use the OCK 2.0 image.

The **acceptance criteria** for this phase being done follows:

1. All nodes in the cluster are running OCK 2.0 images.
2. All Oracle Cloud Native Environment 2.0 CLI functionality can be used.
3. Kubernetes version upgrade is not required if the existing 1.x system has Kubernetes 1.26-1.30.

###  Phase Three: Migration From auth-proxy
This phase is focused on a replacement of auth-proxy to provide OAuth2 functionality.
Fluentd and all the consoles currently require auth-proxy, which relies 
on the Keycloak stack. Because auth-proxy is obsolete, a solution to replace it must
be provided along with instructions on how to migrate to the new solution.

There are other migrations which should be done at some point, such as migration from the existing 
Fluentd Daemonset to Fluent Operator Fluentd.  These migrations are not included in this phase.

The **acceptance criteria** for this phase being done follows:

1. The auth-proxy can be removed from system without breaking supported consoles, logging, and identity.
2. The OAuth2 functionality continues to work using the auth-proxy replacement.

## Obsolete and Unsupported Components
This section lists components that are either obsolete or do not require migration.

### Obsolete components 
These components are obsolete and will be removed from the system during phase one.

* Verrazzano Platform Operator
* Verrazzano Application Operator
* Verrazzano Monitoring Operator
* Verrazzano Cluster Operator
* Verrazzano CAPI controllers
* OAM Kubernetes Runtime

### Unsupported components
These components are unsupported, but not removed.
You may continue to use these components at your own discretion.

* Argo CD
* Coherence
* Rancher
* Thanos

---
[Next: Phase One](./phase1/phase1.md)  
[Previous: README](./README.md)