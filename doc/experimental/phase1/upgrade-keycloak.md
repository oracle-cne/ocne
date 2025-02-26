# Upgrade Keycloak

### Version: v0.0.1-draft

## Export the user supplied overrides of the current release to a file

```text
helm get values -n keycloak keycloak > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/fullnameOverride/{n;N;N;d}' overrides.yaml
```

## Uninstall existing Keycloak deployment
The existing deployment is using a chart name of `keycloakx`. Keycloak will need to be reinstalled to correct the chart name to be `keycloak`.
```text
ocne application uninstall --release keycloak --namespace keycloak
```

## Install Keycloak 21.1.2 using the overrides extracted above:
```text
ocne application install --release keycloak --namespace keycloak --version 21.1.2 --name keycloak --values overrides.yaml
```

## Wait for the installation to complete
```text
kubectl rollout status statefulset --namespace keycloak keycloak -w
```