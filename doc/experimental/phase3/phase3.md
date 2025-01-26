# Phase Three: Unsupported Components that Require Migration

### Version: v0.0.3-draft

## Prerequisites
Set the environment variable KUBECONFIG to point to your cluster.

## Overview
There are a few obsolete or unsupported components that are currently integrated with supported components.
For example, Fluentd and Grafana require auth-proxy.  Likewise, Keycloak, MySQL, and MySQLOperator are not supported,
but are currently required for identity. The goal of this phase is to develop a migration path so that replacement solutions
can be developed and integrated without breaking the current functionality of the system.

### Replace auth-proxy with Dex (or something else)
* auth-proxy (obsolete)

### Components that can be already be upgraded using Helm
* Keycloak
* MySQL
* MySQL Operator

---
[Previous: Phase Two](../phase2/phase2.md)