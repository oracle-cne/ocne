# Migrate to kube-prometheus-stack

### Version: v0.0.1-draft

Verrazzano installs the kube-prometheus-stack using a Helm chart named prometheus-operator.  As a result,
the Helm release is named prometheus-operator and all resources that use {RELEASE-NAME} in the Helm
manifests will have the name prometheus-operator. Consequently, you cannot directly upgrade to the 
version of kube-prometheus-stack that exists in the catalog.  T

In addition, the image section of the overrides file must be modified to use the correct images required
by Oracle Cloud Native Environment.

The purpose of this document is to migrate the Verrazzana kube-prometheus-stack (named prometheus-operator), 
to the catalog kube-prometheus-stack so that future upgrades can be done using the catalog.  

The steps are summarized below:
1. export the Helm user provided overrides to an overrides file
2. modify the image section of the overrides file
3. change promethues PV reclaim policy to detain
4. detach the prometheus PV
5. uninstall the prometheus-operator chart (which is really the kube-prometheus-stack)
6. install kube-state-metrics from the catalog using the override file from step 2 with PVC/PV overrides to use new PV

## Modify the overrides

Export the user supplied overrides of the current release to a file and remove the image overrides:
```text
helm get values -n verrazzano-monitoring prometheus-operator > overrides.yaml
sed -i '1d' overrides.yaml
sed -i '/image: ghcr.io/d' overrides.yaml
sed -i '/alertmanagerDefaultBaseImage:/d' overrides.yaml
sed -i '/alertmanagerDefaultBaseImageRegistry:/d' overrides.yaml
sed -i '/prometheusDefaultBaseImage:/d' overrides.yaml
sed -i '/prometheusDefaultBaseImageRegistry:/d' overrides.yaml

sed -i '/image:/,+3d' overrides.yaml
sed -i '/thanosImage:/,+3d' overrides.yaml
```

Update the existing installation:
```text
ocne application update --release kube-state-metrics --namespace verrazzano-monitoring --version 2.8.2 --reset-values --values overrides.yaml
```
