# Migration from Verrazzano Auth proxy to OAuth2 Proxy

### Version: v0.0.1
This document explains how to migrate from the Verrazzano Auth proxy to the [OAuth2 proxy](https://github.com/oauth2-proxy/oauth2-proxy).


## Install oauth2-proxy
Before installing the OAuth2 Proxy, you must do the following:

1. Create a Keycloak TLS secret.
2. Create a OAuth2 Proxy secret with OIDC credentials.
3. Create a configuration file to be used for installation.
4. Add an email address to tke Keycloak verrazzano-pkce client

### Creating the Keycloak TLS secret
The existing Keycloak TLS secret needs to be copied from the keycloak namespace to verrazzano-system.  This is a simple procedure as described below.

First create a file containing the existing secret:
```
kubectl get secret keycloak-tls -n keycloak -o yaml > ./keycloak-oauth2-tls
```

Next edit ./keycloak-oauth2-tls and do the following:
1. change **namespace: keycloak** to **namespace: verrazzano-system**
2. delete **creationTimestamp** line
3. delete **resourceVersion** line
4. delete **uid** line

Finally, create the new secret
```
kubectl apply -f ./keycloak-oauth2-tls
```

Ensure that the secret has been created
```
kubectl get secret -n verrazzano-system | grep keycloak

---output
keycloak-oauth2-tls                                     kubernetes.io/tls    3      41h
```
