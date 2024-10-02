# Phase Three: Unsupported Components that Require Migration

### Version: v0.0.1-draft

## Overview
Instructions for performing an in-place upgrade of a Kubernetes cluster from Oracle Cloud Native Environment 1.x to 2.x.

There are a few obsolete or unsupported components that are currently integrated with supported components.
For example, Fluentd and Grafana require auth-proxy.  Likewise, Keycloak, MySQL, and MySQLOperator are required
for identity.  This goal of this phase is to develop a migration path so that replacement solutions
can be developed and integrated without breaking the current functionality of the system.

The lists below summarize the components that need a migration solution:

### Migrate Opensearch and OpenSearch Dashboard to be managed by Helm
Because both the Verrazzano Platoform operator and Monitoring operator have been removed, there is no way to upgrade the following components:
* OpenSearch
* OpenSearch Dashboard

There needs to be a migration path to have these components managed by a Helm chart.

### Replace auth-proxy with Dex
* auth-proxy (obsolete)

### Components that can be upgraded using Helm
* Keycloak
* MySQL
* MySQL Operator