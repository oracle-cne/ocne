# Application Management

Applications in Kubernetes clusters are managed using the `ocne application`
command and its subcommands.  Each application is installed from a catalog.
Catalogs are managed using the `ocne catalog` command and its subcommands.

## Catalogs

Catalogs are sources of applications.  Search a catalog to find what
applications it contains.

### Defining Catalogs

A catalog is a Kubernetes Service resource with particular annotations and
labels.  The CLI and UI inspect these services and use the annotations to
interpret how to interact with that catalog.

The following annotations are used:
| Annotation                 | Use |
|----------------------------|-----|
| catalog.ocne.io/name       | The name of the catalog as it appears in the CLI and UI |
| catalog.ocne.io/uri        | An optional value that contains any relative path information required to access a catalog |
| catalog.ocne.io/protocol   | The protocol used by the catalog.  Valid values are "helm" and "artifacthub" |

| Labels                     | Use |
|----------------------------|-----|
| catalog.ocne.io/is-catalog | If this annotation is present, the Service is treated as a Catalog |

The Service for the Oracle Cloud Native Environment Application Catalog looks
like this:
```
apiVersion: v1
kind: Service
metadata:
  annotations:
    catalog.ocne.io/name: Oracle Cloud Native Environment Application Catalog
    catalog.ocne.io/protocol: helm
  labels:
    catalog.ocne.io/is-catalog: ""
  name: app-catalog
  namespace: ocne-system
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/instance: app-catalog
    app.kubernetes.io/name: app-catalog
  type: NodePort
```

### Listing Catalogs

The set of available catalogs can be listed.

```
$ ocne catalog list
CATALOG                                            	NAMESPACE  	PROTOCOL	URI
Oracle Cloud Native Environment Application Catalog	ocne-system	helm    	   
embedded                                           	           	helm
```

### Adding Catalogs

Catalogs can be added to a cluster.  Once a catalog has been added, the contents
of the catalog can be searched, installed, used to generate configuration
templates, and other application management actions.

There is support for two kinds of catalogs: regular helm chart repositories and
ArtifactHub.  The type of catalog must be specified when adding a catalog to
a cluster.

#### Adding a Catalog via the CLI

```
# Add a catalog that points to artifacthub.io
$ ocne catalog add -p artifacthub -n artifacthub -u https://artifacthub.io

# Once the catalog has been added to the cluster, it appears in the list
$ ocne catalog list
CATALOG                                            	NAMESPACE  	PROTOCOL   	URI                   
Oracle Cloud Native Environment Application Catalog	ocne-system	helm       	                      
artifacthub                                        	ocne-system	artifacthub	https://artifacthub.io
embedded                                           	           	helm
```

#### Adding a Catalog Manually

Catalogs can be defined by adding Service resources directly to a Kubernetes
cluster.  This can be done manually with `kubectl` or a related tool, or using
a resource manager like `helm`.

```
$ kubectl apply -n mycatalog-namespace -f - << EOF
apiVersion: v1
kind: Service
metadata:
  annotations:
    catalog.ocne.io/name: My Custom Catalog
    catalog.ocne.io/protocol: helm
  labels:
    catalog.ocne.io/is-catalog: ""
  name: mycatalog
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/instance: mycatalog
    app.kubernetes.io/name: mycatalog
  type: ClusterIP
EOF

$ ocne catalog list
CATALOG                                            	NAMESPACE          	PROTOCOL	URI
My Custom Catalog                                  	mycatalog-namespace	helm    	   
Oracle Cloud Native Environment Application Catalog	ocne-system        	helm    	   
embedded                                           	           	        helm
```

### Removing Catalogs

Any catalogs that have been added can be removed.  Removing a catalog does
not uninstall applications from that catalog.  Applications installed from
the catalog can be uninstalled even after the catalog has been removed.

#### Removing a Catalog via the CLI

```
$ ocne catalog remove -n artifacthub

$ ocne catalog list
CATALOG                                            	NAMESPACE  	PROTOCOL	URI
Oracle Cloud Native Environment Application Catalog	ocne-system	helm
embedded                                           	           	helm
```

#### Removing a Catalog Manually

Catalogs can also be removed by deleting the Service that defines the catalog.
This can be done by deleting the Service manually or by using a resource
manager like `helm`.

```
$ kubectl delete -n mycatalog-namespace service mycatalog
```

### Finding Applications in a Catalog

The complete contents of an application can be seen by searching a catalog
without specifying any search parameters

```
$ ocne catalog search
APPLICATION             	VERSION
ocne-catalog             	2.0.0  
cert-manager            	v1.1.3 
cert-manager-webhook-oci	0.1.0  
flannel                 	v0.22.3
fluent-operator         	2.5.0  
grafana                 	7.5.5 
ui                              2.0.0
kube-prometheus-stack   	45.25.0
kube-state-metrics      	2.8.2
prometheus-adapter      	4.2.0  
prometheus-node-exporter	4.23.2 
```

To find a specific application, use a search pattern

```
$ ocne catalog search --pattern 'grafana'
APPLICATION	VERSION
grafana    	7.5.5


$ ocne catalog search --pattern 'graf*'
APPLICATION	VERSION
grafana    	7.5.5
```

To search an alternate catalog, specify its name.

Search the catalog named `embedded`, which is built in the CLI.
```
$ ocne catalog search --name embedded
```

Search the catalog named `artifacthub`.
```
$ ocne catalog search --name artifacthub --pattern ingress-nginx
APPLICATION               	VERSION
ingress-nginx             	4.0.18 
ingress-nginx             	1.0.0  
ingress-nginx             	4.10.1 
ingress-nginx             	4.10.1 
ingress-nginx             	4.9.1  
ingress-nginx             	4.8.4  
ingress-nginx             	4.10.1 
ingress-nginx             	4.5.2  
ingress-nginx             	4.0.13 
ingress-nginx             	4.1.0  
ingress-nginx             	4.10.1 
ingress-nginx             	3.29.1 
ingress-nginx             	4.7.0  
ingress-nginx             	4.5.2  
ingress-nginx-external-lb 	1.0.0  
ingress-nginx-monitoring  	1.2.3  
ingress-nginx-validate-jwt	1.13.46
```

Some catalogs have too many results to do an unqualified search.. ArtifactHub
has over 10,000 charts.

```
$ ocne catalog search --name artifacthub                                  
ERRO[2024-05-08T13:17:30-05:00] ArtifactHub powered catalogs do not support unqualified searches
```

## Applications

Applications can be installed, uninstalled, and viewed.

### Listing Installed Applications

The set of installed applications can be listed.

```
$ ocne application list -a
Releases
NAME       	NAMESPACE   	CHART      	STATUS  	REVISION	APPVERSION
app-catalog	ocne-system 	app-catalog	deployed	1       	2.0.0    
flannel    	kube-flannel	flannel    	deployed	1       	0.22.3   
grafana    	grafana     	grafana    	deployed	1       	7.5.5     
ui              ocne-system     ui              deployed	1       	2.0.0
```

### Viewing an Application

The details of an application can be viewed.

```
$ ocne application show --namespace kube-flannel --release flannel -c
flannel:
  args:
  - --ip-masq
  - --kube-subnet-mgr
  backend: vxlan
  image:
    repository: container-registry.oracle.com/olcne/flannel
    tag: v0.22.3-1
  image_cni:
    repository: null
    tag: null
podCidr: 10.244.0.0/16
podCidrv6: ""
----------------
NAME: flannel
NAMESPACE: kube-flannel
CHART: flannel
STATUS: deployed
REVISION: 1
APPVERSION: v0.22.3
```

### Configuring Applications

The configuration options for an application can be extracted from the catalog
and viewed, saved to a file, or edited directly.

To view the configuration, generate a template.  To save the template
to a file to use for an installation, redirect the output to a file.

```
$ ocne application template --name grafana | less
rbac:
  create: true
  ## Use an existing ClusterRole/Role (depending on rbac.namespaced false/true)
  # useExistingRole: name-of-some-(cluster)role
  pspEnabled: true
  pspUseAppArmor: true
  namespaced: false
  extraRoleRules: []
  # - apiGroups: []
  #   resources: []
...
```

You can view the configuration of an application using the catalog built into the CLI. This can be done without a running cluster.
```
$ ocne application template --name grafana --catalog embedded | less
rbac:
  create: true
  ## Use an existing ClusterRole/Role (depending on rbac.namespaced false/true)
  # useExistingRole: name-of-some-(cluster)role
  pspEnabled: true
  pspUseAppArmor: true
  namespaced: false
  extraRoleRules: []
  # - apiGroups: []
  #   resources: []
...
```

Templates can be edited directly if a default editor is configured.

```
# The EDITOR environment variable is used to specify the editor to use
$ EDITOR=vim ocne application template --name grafana --interactive

# The resulting template is written to disk
$ ls
grafana-values.yaml
```

### Installing Applications

Applications are installed from catalogs.  By default, the Oracle Cloud Native
Environment Catalog is the source of applications.  Applications may also be installed from the catalog named `embedded`, which is built into the CLI binary.

Many applications can be installed multiple times.  Each unique installation of an application 
is known as a "release".  Releases are installed into particular Kubernetes namespaces.

```
# Install the application
$ ocne application install --release prometheus --namespace prometheus --name prometheus

# It is now visible in the list
$ ocne application list --all
Releases
NAME       	NAMESPACE   	CHART      	STATUS  	REVISION	APPVERSION
app-catalog	ocne-system 	app-catalog	deployed	1       	2.0.0    
flannel    	kube-flannel	flannel    	deployed	1       	0.22.3   
ui              ocne-system 	ui              deployed	1       	2.0.0
prometheus 	prometheus  	prometheus 	deployed	3       	2.31.1 

ocne application list --namespace prometheus
Releases
NAME      	NAMESPACE 	CHART     	STATUS  	REVISION	APPVERSION
prometheus	prometheus	prometheus	deployed	3       	2.31.1

$ kubectl -n prometheus get pods
NAME                                READY   STATUS    RESTARTS   AGE
prometheus-server-d899755c4-9p7tz   1/2     Running   0          13s
```

Install Grafana from the catalog named `embedded`, which is built into the CLI binary.
```
$ ocne application install --name grafana --release grafana --catalog embedded --namespace grafana

# It is now visible in the list
$ ocne application list --namespace grafana

NAME   	NAMESPACE	CHART  	STATUS  	REVISION	APPVERSION
grafana	grafana  	grafana	deployed	2       	7.5.17    
```

Applications can also be customized via a values file during installation
```
# Install prometheus with node-exporter enabled
$ ocne application install --release prometheus --namespace prometheus --name prometheus --values - << EOF
serviceAccounts:
  nodeExporter:
    create: true
    name:
    annotations: {}

nodeExporter:
  enabled: true
  image:
    repository: container-registry.oracle.com/verrazzano/node-exporter
    tag: v1.3.1
    pullPolicy: IfNotPresent
EOF

# Notice that node-exporter is now running as well
$ kubectl -n prometheus get pods
NAME                                READY   STATUS    RESTARTS   AGE
prometheus-node-exporter-cnt7r      1/1     Running   0          8m6s
prometheus-server-d899755c4-svq99   2/2     Running   0          8m6s
```

### Uninstalling Applications

Applications can be uninstalled.

```
# Uninstall the application
$ ocne application uninstall --release prometheus --namespace prometheus
INFO[2024-07-10T19:26:35Z] Uninstalling release prometheus
INFO[2024-07-10T19:26:35Z] prometheus uninstalled successfully

# It is no longer in the list
$ ocne application list -a
NAME       	NAMESPACE   	CHART      	STATUS  	REVISION	APPVERSION
app-catalog	ocne-system 	app-catalog	deployed	1       	2.0.0    
flannel    	kube-flannel	flannel    	deployed	1       	0.22.3   
ui              ocne-system 	ui              deployed	1       	2.0.0

# The pods are gone
$ kubectl -n prometheus get pods
No resources found in prometheus namespace.
```

### Updating Applications

Installed applications can be updated.  The same command is used to update
configurations and versions.  The configuration and version can be updated
simultaneously.

```
# Create a default installation of Prometheus
$ ocne application install --release prometheus --namespace prometheus --name prometheus
INFO[2024-07-10T19:28:45Z] Application installed successfully

$ kubectl -n prometheus get pods
NAME                                READY   STATUS    RESTARTS   AGE
prometheus-server-d899755c4-9p7tz   1/2     Running   0          13s

# Update the application to include node-exporter
$ ocne application update --release prometheus --namespace prometheus --values - << EOF
serviceAccounts:
  nodeExporter:
    create: true
    name:
    annotations: {}

nodeExporter:
  enabled: true
  image:
    repository: container-registry.oracle.com/verrazzano/node-exporter
    tag: v1.3.1
    pullPolicy: IfNotPresent
EOF

$ kubectl -n prometheus get pods
NAME                                READY   STATUS    RESTARTS   AGE
prometheus-node-exporter-4fqqc      1/1     Running   0          8s
prometheus-server-d899755c4-9p7tz   2/2     Running   0          60s
```

Update the application Grafana from the catalog named `embedded`, which is built in the CLI.
```
$ ocne application update --release grafana --catalog embedded --namespace grafana

# It is now visible in the list
$ ocne application list --namespace grafana

NAME   	NAMESPACE	CHART  	STATUS  	REVISION	APPVERSION
grafana	grafana  	grafana	deployed	2       	7.5.17    
```
