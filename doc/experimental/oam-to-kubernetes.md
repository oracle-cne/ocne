# OAM to Kubernetes Mappings

### Version: v0.0.1-draft

## Verrazzano CLI export command
Verrazzano provides a CLI command that you can use to facilitate the migration of an OAM application to be managed as a collection of Kubernetes objects.

The command vz export oam will output the YAML for each Kubernetes object that was generated as a result of deploying an OAM application. The generated YAML is sanitized so that it can be used to deploy the application. Fields such as creationTimestamp, resourceVersion, uid, and status are not included in the output.

For example, the following CLI command exports the YAML from the hello-helidon OAM sample application.

```text
vz export oam --name hello-helidon --namespace hello-helidon > myapp.yaml
```

You can use the output of the command vz export oam to deploy the application on another cluster.

```text
kubectl create namespace hello-helidon
kubectl apply -f myapp.yaml
```

In addition, you can edit the output of the command vz export oam before deploying the application. The extent to which the exported YAML may be edited will vary based on local requirements. Here are some examples of changes that may be made to the exported YAML:

* The Kubernetes namespace of where to deploy the application
* Add or modify labels or annotations on objects
* Port assignments
* Authorization policies
* Values for secrets
* Mount volume definitions
* Replica counts
* Prometheus logging rules


