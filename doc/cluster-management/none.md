# None Provider

In some cases, it is necessary to deploy the Oracle Linux Cloud Native Environment 2.0 
app-catalog and UI into an existing Kubernetes cluster. 


## Prerequisites

A kubeconfig pointing to a active and healthy Kubernetes cluster 
is required to use the none provider.

## Using the None Provider 

An existing 1.x cluster can be configured using the provider by starting a cluster
with the provider set to "none" and passing in a kubeconfig. An example is shown below 

```
ocne cluster start --provider none -k ~/.kube/kubeconfig.test-cluster
INFO[2024-08-19T15:20:34Z] Installing ui into ocne-system: ok 
INFO[2024-08-19T15:20:35Z] Installing app-catalog into ocne-system: ok 
INFO[2024-08-19T15:20:35Z] Kubernetes cluster was created successfully  
INFO[2024-08-19T15:20:55Z] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/home/opc/.kube/kubeconfig.test-cluster' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y               

INFO[2024-08-19T15:21:06Z] Post install information:


To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system 

```
