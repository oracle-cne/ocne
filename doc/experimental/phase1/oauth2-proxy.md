# Migration from Verrazzano Auth proxy to OAuth2 Proxy

### Version: v0.0.1
This document explains how to migrate from the Verrazzano Auth proxy to the [OAuth2 proxy](https://github.com/oauth2-proxy/oauth2-proxy).


## Install oauth2-proxy
Before installing the OAuth2 Proxy, you must do the following:

1. Create a Keycloak TLS secret.
2. Create a OAuth2 Proxy secret with OIDC credentials.
3. Create a configuration file to be used for installation.
4. Add an email address to tke Keycloak verrazzano-pkce client

### Create the Keycloak TLS secret
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

### Create the OAuth2 Proxy credentials secret
The oauth2-proxy pod refers to a secret which contains the following fields:

1. client-id
2. client-secret
3. cookie-secret

#### create the client id
The client id, already exists in Keycloak so this will be used.  You can get if from 
```
echo -n  | base64

---output
```

#### create the client secret
The client secret is required by the OAuth2 Proxy even though it is not used in this case.  Generate a fake secret using any string, for example:

```
echo -n fake-secret | base64

---output
ZmFrZS1zZWNyZXQ=
```

#### create the cookie secret
The cookie secret is a binary 32 byte value that must be base64-URL encoded, then that output needs to be base64 encoded again.

```
openssl rand  32  | base64 | tr '/+' '_-' | tr -d '=' | base64

---output
NXBGUk11aU1vOXB5dV...
```

### create and apply secret.yaml file
Create a secret YAML file with the values from the first 3 steps (replace the <...> sections with real values).

```
apiVersion: v1
data:
  cookie-secret: <cookie-secret>
  client-id: <client-id>
  client-secret: <client-secret>
kind: Secret
metadata:
  name: oauth2-proxy
  namespace: verrazzano-system
type: Opaque
```

