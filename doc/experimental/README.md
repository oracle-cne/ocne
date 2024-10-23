# Migration to Oracle Cloud Native Environment 2.0

### Version: v0.0.4-draft
This set of documents contains instructions to migrate from an Oracle Cloud Native Environment 1.x with Verrazzano installed 
to an Oracle Cloud Native Environment 2.0.

## Distinction between migration and upgrade
There is a clear distinction between migration from Verrazzano with Oracle Cloud Native Environment 1.x
and upgrades. It is very important to understand this. The migration is a one time operation that only upgrades 
a few components as needed, whereas upgrade in general (post-migration) is an ongoing system management user responsibility. 
This include OCK upgrades to get newer Kubernetes versions and application (product) upgrades from the catalog.
Migration does NOT include changes to Kubernetes version. If your existing Kubernetes version is 
earlier than 1.26, then you need to upgrade to 1.25 before migrating to OCK.

Once you have migrated, all future upgrades of Kubernetes are done via the `ocne cluster stage` and `ocne node update` commands.
Likewise, application upgrades are done through the catalog. The coordination of upgrading the catalog applications vs Kubernetes, 
and making sure that versions are compatible is your responsibility.
Oracle provides [documentation](https://docs.oracle.com/en/operating-systems/olcne/2.0/clusters/admin.html) on how to 
update OCK to get newer versions of Kubernetes. 

**NOTE:** These instructions are in the experimental phase and **MUST** not be run against a production environment. 

**NOTE**: These instructions **MUST** be performed in the sequence outlined in these documents, starting with the Introduction, then Phase One, etc.


## [Introduction](./introduction.md)

## [Phase One: Verrazzano Migration](phase1/phase1.md)

## [Phase Two: Oracle Cloud Native Environment 2.0 OCK Migration](phase2/phase2.md)

## [Phase Three: Unsupported Components that Require Migration](phase3/phase3.md)

---
[Next: Introduction](./introduction.md)