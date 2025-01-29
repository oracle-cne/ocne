# Migration from Verrazzano Auth proxy to OAuth2 Proxy

### Version: v0.0.1-draft
This document explains how to migrate from the Verrazzano Auth proxy to the [OAuth2 proxy](https://github.com/oauth2-proxy/oauth2-proxy).

## Summary of steps
1. Prepare for installation of OAuth2 Proxy
2. Install OAuth2 Proxy
3. Disable Verrazzano auth proxy and Verrazzano monitoring operator
4. NGINX ingress controller configuration changes
5. Migrate each console to use OAuth2 Proxy
6. Update Fluentd config 
7. Remove Verrazzano auth proxy from the cluster

## Prepare for installation of OAuth2 Proxy 
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
The client id, already exists in Keycloak so this will be used.  You can get the clear text client id as follows:
```
 helm --kubeconfig paul-kubeconfig get values -n verrazzano-system verrazzano-authproxy | grep OIDCClientID
 
 ---output
  OIDCClientID: <client-id>
```

Next, base64 encode the client id.  Replace the <...> section with real value from the previous command:
```
echo -n  <client-id> | base64

---output
<client-id-base64>
```

#### create the client secret
The client secret is required by the OAuth2 Proxy even though it is not used in this case.  Generate a fake secret using any string, for example:

```
echo -n fake-secret | base64

---output
<client-secret-base64>
```

#### create the cookie secret
The cookie secret is a binary 32 byte value that must be base64-URL encoded, then that output needs to be base64 encoded again.

```
openssl rand  32  | base64 | tr '/+' '_-' | tr -d '=' | base64

---output
<cookie-secret-base64>
```

### create and apply secret.yaml file
Create a secret YAML file, name oauth2-proxy.yaml, with the values from the first 3 steps (replace the <...> sections with real values).

```
apiVersion: v1
data:
  cookie-secret: <cookie-secret-base64>
  client-id: <client-id-base64>
  client-secret: <client-secret-base64>
kind: Secret
metadata:
  name: oauth2-proxy
  namespace: verrazzano-system
type: Opaque
```

Apply the secret file to create the secret
```
kubectl apply -f oauth2-proxy.yaml
```

Ensure that the secret has been created
```
kubectl get secret -n verrazzano-system oauth2-proxy

---output
NAME           TYPE     DATA   AGE
oauth2-proxy   Opaque   3      3d20h
```

### Create OAuth2 Proxy overrides file
Create an overrides file to be used when installing oauth2-proxy from the catalog.
Define the keycloak URL in an environment variable, replacing the <...> section with the real URL.:
```
KEYCLOAK_URL=<keycloak_url>
```

Execute the follwoing command to generate the oauth2-proxy overrides file.
```text
cat <<EOF > ./oauth2-values.yaml 
extraVolumes:
  - name: keycloak-ca-bundle-cert
    secret:
      secretName: keycloak-oauth2-tls

extraVolumeMounts:
  - mountPath: /etc/ssl/certs/keycloak/
    name: keycloak-ca-bundle-cert

customLabels:
  sidecar.istio.io/inject: "false"
service:  
  portNumber: 49000
config:
  existingSecret: oauth2-proxy
  configFile: |-
    provider_ca_files = [ "/etc/ssl/certs/keycloak/tls.crt", "/etc/ssl/certs/keycloak/ca.crt" ]
    email_domains = [ "*" ]
    insecure_oidc_allow_unverified_email = true
    upstreams="file:///dev/null"
    provider = "oidc"
    code_challenge_method = "S256"
    oidc_issuer_url = "$KEYCLOAK_URL/auth/realms/verrazzano-system"
    skip_provider_button = true
    approval_prompt = "auto"
    reverse_proxy = true
    set_xauthrequest = true
    set_authorization_header = true
    pass_user_headers = true
    pass_access_token = true
EOF    
```


### Add email to the Keycloak client
Log into Keycloak admin console and add an email to the client using the following steps.

1. Get the Keycloak URL:
```
vz status

---output
Verrazzano Status
...
  Access Endpoints:
...
    keyCloakUrl: <keycloak-url>
```

2. Get the **keycloakadmin** password:
```
kubectl get secret \
    --namespace keycloak keycloak-http \
    -o jsonpath={.data.password} | base64 \
    --decode; echo
```

3. Navigate to the keycloak URL in a browser, and log into Keycloak as user ***keycloakadmin**. Select **Clients** in the left navigation pane and the list of clients will be shown in the middle pane
under the heading **Client ID**.  Select the correct client then update the email with a valid email and click the Save button.

## Install OAuth2 Proxy
