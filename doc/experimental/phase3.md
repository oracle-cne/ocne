# Phase Three: Unsupported Components that Require Migration

### Version: v0.0.1-draft

## Overview
There are a few obsolete or unsupported components that are currently integrated with supported components.
For example, Fluentd and Grafana require auth-proxy.  Likewise, Keycloak, MySQL, and MySQLOperator are not supported,
but are currently required for identity. The goal of this phase is to develop a migration path so that replacement solutions
can be developed and integrated without breaking the current functionality of the system.

### Migrate Opensearch and OpenSearch Dashboard to be managed by Helm
Because both the Verrazzano Platoform operator and Monitoring operator have been removed, there is no way to upgrade the following components:
* OpenSearch
* OpenSearch Dashboard

There needs to be a migration path to have these components managed by a Helm chart.

### Replace auth-proxy with Dex (or something else)
* auth-proxy (obsolete)

### Components that can be already be upgraded using Helm
* Keycloak
* MySQL
* MySQL Operator