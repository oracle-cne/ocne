// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vsphere

import (
    "errors"
    "fmt"
    "github.com/oracle-cne/ocne/pkg/cluster/template"
    "github.com/oracle-cne/ocne/pkg/config/types"
    "github.com/oracle-cne/ocne/pkg/util"
)

// Minimal input struct for templating. We keep it close to existing providers for consistency.
// Adjust/extend as needed once cluster definition schema for vsphere is finalized.
type vsphereData struct {
    ClusterConfig *types.ClusterConfig
    // Basic values passed through
    CipherSuite   string
    VolumePluginDir string
}

// GetVsphereTemplate renders the vSphere CAPI template that specifies the CAPI resources.
// It mirrors the OCI/OLVM flow: load embedded template, validate, then execute with data.
func GetVsphereTemplate(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
    tmplBytes, err := template.ReadTemplate("capi-vsphere.yaml")
    if err != nil {
        return "", err
    }

    if clusterConfig.ControlPlaneNodes < 1 {
        return "", errors.New("the number of control plane nodes must be at least 1")
    }
    if clusterConfig.ControlPlaneNodes%2 == 0 {
        return "", errors.New("the number of control plane nodes must be odd")
    }

    if clusterConfig.KubeVersion == "" {
        return "", errors.New("kubernetesVersion is required")
    }
    if clusterConfig.KubeAPIServerBindPort == 0 {
        return "", errors.New("kubeApiServerBindPort must be greater than 0")
    }

    // Basic required fields for vsphere provider
    vp := clusterConfig.Providers.Vsphere
    if vp.Namespace == "" {
        return "", errors.New("vsphere provider requires namespace")
    }
    if vp.Server == "" || vp.Datacenter == "" || vp.Network == "" || vp.Datastore == "" || vp.ResourcePool == "" || vp.Template == "" {
        return "", fmt.Errorf("vsphere provider requires server, datacenter, network, datastore, resourcePool, template")
    }
    if vp.Username == "" || vp.Password == "" {
        return "", fmt.Errorf("vsphere provider requires credentials")
    }
    if vp.ControlPlaneEndpoint == "" {
        return "", fmt.Errorf("vsphere provider requires controlPlaneEndpoint")
    }

    data := &vsphereData{
        ClusterConfig:   clusterConfig,
        CipherSuite:     clusterConfig.CipherSuites,
        VolumePluginDir: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec", // conventional; adjust if needed
    }

    return util.TemplateToStringWithFuncs(string(tmplBytes), data, nil)
}
