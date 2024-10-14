# Remove OAM objects

### Version: v0.0.1-draft
This document explains how to remove the OAM objects.

***WARNING*** Before following these steps, you MUST complete the steps to remove Verrazzano controllers from the system, 
see [instructions](../phase1/disable-verrazzano.md).  If you proceed without doing that your application WILL get deleted.

## Installing Verrazzano 1.6.11 CLI on a Linux AMD64 machine
Verrazzano provides a CLI command that you can use to facilitate the migration of an OAM application to be managed as a collection of Kubernetes objects.
If you don't have the 1.6.11 CLI (or later), you need to download it, then use it to export the Kubernetes objects that were generated
for an OAM applications to Kubernetes YAML manifests.

These instructions demonstrate installing the CLI on a Linux AMD64 machine:
```
curl -LO https://github.com/verrazzano/verrazzano/releases/download/v1.6.11/verrazzano-1.6.11-linux-amd64.tar.gz
tar xvf verrazzano-1.6.11-linux-amd64.tar.gz
sudo cp verrazzano-1.6.11/bin/vz /usr/local/bin
```

Check the version:
```
vz version

Version: v1.6.11
BuildDate: 2024-02-29T20:12:32Z
GitCommit: 1a31fa57c75d570220674b03f5297130bcecb311
```