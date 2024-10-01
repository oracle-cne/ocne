# Introduction

### Version: v0.0.1-draft

## Migration Summary

Migration is divided into three main phases:
1. Verrazzano migration
2. Oracle Cloud Native Environment 2.0 OCK migration
3. Migration from unsupported components.

### Phase One: Verrazzano Migration
Verrazzano migration means the customer is no longer using Verrazzano to manage the system and applications When Verrazzano migration is complete, all application lifecycle management will be done via helm directly, using the Oracle Cloud Native Environment catalog, other catalogs, various Helm charts, etc. At this point Verrazzano is effectively removed from the system and OAM resources no longer exist.

The goal of this phase is to stop using Verrazzano and start using the Oracle Cloud Native Environment 2.0 Catalog for component/application life cycle management.  Nothing in the topology or system architecture changes during this phase.

The acceptance criteria for this phase being done follows:

1. Oracle Cloud Native Environment 2.0 catalog and UI installed on the cluster
2. Verrazzano controllers and related resources are removed from cluster
3. The life cycle of all supported components can be done via the Oracle Cloud Native environment 2.0 catalog
4. The installed versions of all components can be used, they do not need to be upgraded
5. **?? Not in this phase??** OAM resources translated to Kubernetes native resource YAML
    - Existing cluster resources are not affected
6. Some Oracle Cloud Native Environment 2.0 CLI functionality is not yet available (stage, update, etc.)

### Phase Two: Oracle Cloud Native Environment 2.0 OCK Migration
Oracle Cloud Native Environment 2.0 Ock migration means that all the Kubernetes hosts are running OCK images with Oracle Cloud Native Environment 2.0, instead of 1.7. This phase will require in-place migration where no new nodes are added, rather existing nodes are updated to use the OCK 2.0 image.

###  Phase Three: Migration from unsupported components

There are a few components currently required by the customer that will no longer be supported, some were developed by the Verrazzano team, and others are third party.  The lists below summarize the components, other sections of this document will have migration plans.

**Verrazzano Owned** (These components were developed by Verrazzano)
* auth-proxy

**Third Party**
* OpenSearch
* Keycloak
* MySQL
* MySQL Operator

**Issues**
* Istio Authorization Polices
* Network Policies

## Obsolete components
Obsolete components consist of components that we neither support nor have migration solutions, such as Rancher.  The customer may or may not continue to use these components, but Oracle is not involved.

**Verrazzano Owned** (These components were developed by Verrazzano)
* Verrazzano Platform Operator
* verrazzano Application Operator
* Verrazzano Cluster Operator
* Verrazzano CAPI controllers

**Third Party** (completely unsupported)
* Argo
* Coherence
* Rancher
* Thanos