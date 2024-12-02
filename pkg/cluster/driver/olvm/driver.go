// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/kubepki"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"os"
	"path/filepath"
)

const (
	DriverName       = "olvm"
	ClusterNameLabel = "cluster.x-k8s.io/cluster-name"
	EnvUsername      = "OCNE_OLVM_USERNAME"
	EnvPassword      = "OCNE_OLVM_PASSWORD"
	EnvScope         = "OCNE_OLVM_SCOPE"
)

type OlvmDriver struct {
	Ephemeral            bool
	BootstrapKubeConfig  string
	KubeConfig           string
	Config               *types.Config
	ClusterConfig        *types.ClusterConfig
	ClusterResources     string
	PKIInfo              *kubepki.PKIInfo
	UploadCertificateKey string
	ResourceNamespace    string
	FromTemplate         bool
	Deleted              bool
}

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
	var err error
	doTemplate := false
	cd := clusterConfig.ClusterDefinition
	cdi := clusterConfig.ClusterDefinitionInline
	if cd != "" && cdi != "" {
		// Can't mix inline and file-based resources
		return nil, fmt.Errorf("cluster configuration has file-based and inline resources")
	} else if cd == "" && cdi == "" {
		// If no configuration is provided, make one.  We may need to upload an
		// image.
		doTemplate = true

	} else if cd != "" {
		// If the path to the cluster definition is not
		// absolute, then assume it is relative to the
		// cluster config working directory.
		if !filepath.IsAbs(cd) {
			cd = filepath.Join(clusterConfig.WorkingDirectory, cd)
			cd, err = filepath.Abs(cd)
			if err != nil {
				return nil, err
			}
		}
		cdiBytes, err := os.ReadFile(cd)
		if err != nil {
			return nil, err
		}
		cdi = string(cdiBytes)
	}

	// Unlike other cluster drivers, it is not feasible to have zero
	// worker nodes.  Cluster API will not create control plane nodes
	// with taints removed, and it can get upset if they are removed.
	// Require at least one.
	//
	// If someone really wants to have no workers, then they are free
	// to pass in a cluster definition.
	if clusterConfig.WorkerNodes == 0 {
		clusterConfig.WorkerNodes = 1
	}

	// It's also not feasible to have zero control plane nodes.
	if clusterConfig.ControlPlaneNodes == 0 {
		clusterConfig.ControlPlaneNodes = 1
	}

	cad := &OlvmDriver{
		Config:           config,
		ClusterConfig:    clusterConfig,
		ClusterResources: cdi,
		FromTemplate:     doTemplate,
	}
	bootstrapKubeConfig, isEphemeral, err := start.EnsureCluster(config.Providers.Olvm.KubeConfigPath, config, clusterConfig)
	if err != nil {
		return nil, err
	}

	cad.Ephemeral = isEphemeral
	cad.BootstrapKubeConfig = bootstrapKubeConfig

	_, kubeClient, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return nil, err
	}

	// Install any necessary components into the admin cluster
	capiApplications, err := cad.getApplications()
	if err != nil {
		return nil, err
	}

	err = install.InstallApplications(capiApplications, cad.BootstrapKubeConfig, config.Quiet)
	if err != nil {
		return nil, err
	}

	// Wait for all controllers to come online.  This is done
	// as a separate step so that all the image pulls can happen
	// in parallel because the application installation is
	// linear
	err = cad.waitForControllers(kubeClient)
	if err != nil {
		return nil, err
	}

	cad.KubeConfig, err = client.GetKubeconfigPath(fmt.Sprintf("kubeconfig.%s", cad.ClusterConfig.Name))
	if err != nil {
		return nil, err
	}

	return cad, nil
}
