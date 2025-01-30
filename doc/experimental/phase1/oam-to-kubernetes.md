# OAM to Kubernetes Mappings

### Version: v0.0.4-draft
This document explains how to generate Kubernetes YAML files from OAM-related objects that are running in a Kubernetes cluster.

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

## Run Verrazzano CLI export command
The command vz export oam will output the YAML for each Kubernetes object that was generated as a result of deploying an OAM application.  
The generated YAML is sanitized so that it can be used to deploy the application.  
Fields such as creationTimestamp, resourceVersion, uid, and status are not included in the output.

For example, the following CLI command exports the YAML from the ToDo List OAM sample application,
described at https://verrazzano.io/latest/docs/examples/wls-coh/todo-list/

In this case the ApplicationConfiguration is todo-appconf.
```text
kubectl get appconfig -A
NAMESPACE   NAME           AGE
todo-list   todo-appconf   160m
```

Now generate the Kubernetes manifests that comprise this application and save them in todo.yaml:
```text
vz export oam --name todo-appconf --namespace todo-list > todo.yaml
```

You can use the output of the command vz export oam to deploy the application on another cluster.
In addition, you can edit the todo.yaml to change any manifiest before deploying the application. 
The extent to which the exported YAML may be edited will vary based on local requirements. 
Here are some examples of changes that may be made to the exported YAML:

* The Kubernetes namespace of where to deploy the application
* Add or modify labels or annotations on objects
* Port assignments
* Authorization policies
* Values for secrets
* Mount volume definitions
* Replica counts
* Prometheus logging rules


