// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/config/types"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/spf13/cobra"
)

const (
	CommandName = "start"
	helpShort   = "Start an OCNE cluster"
	helpLong    = `Deploy a cluster from a given configuration. There are four primary flavors of deployments: local virtualization, installation on to pre-provisioned compute resources, 
installation on to self-provisioned compute resources, and those that leverage a cloud provider or other infrastructure automation. Starting a cluster, which is already
running, does not update the installed applications and catalogs.`
	helpExample = `
ocne cluster start --config ~/example-path/config-file --control-plane-nodes 2 --worker-nodes 3
`
)

var config types.Config
var clusterConfig types.ClusterConfig
var clusterConfigPath string

const (
	flagControlPlaneNodes      = "control-plane-nodes"
	flagControlPlaneNodesShort = "n"
	flagControlPlaneNodesHelp  = "The number of control plane nodes to provision for clusters deployed using local virtualization"

	flagWorkerNodes      = "worker-nodes"
	flagWorkerNodesShort = "w"
	flagWorkerNodesHelp  = "The number of worker nodes to provision for clusters deployed using local virtualization"

	flagVirtualIP     = "virtual-ip"
	flagVirtualIPHelp = "The virtual IP address to use as the IP address of the Kubernetes API server"

	flagLoadBalancer     = "load-balancer"
	flagLoadBalancerHelp = "The hostname or IP address of the external load balancer for the Kubernetes API server"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: helpShort,
		Long:  helpLong,
		Args:  cobra.MatchAll(cobra.ExactArgs(0), cobra.OnlyValidArgs),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.Flags().StringVarP(&config.KubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&clusterConfigPath, constants.FlagConfig, constants.FlagConfigShort, "", constants.FlagConfigHelp)
	cmd.Flags().Uint16VarP(&clusterConfig.ControlPlaneNodes, flagControlPlaneNodes, flagControlPlaneNodesShort, 0, flagControlPlaneNodesHelp)
	cmd.Flags().Uint16VarP(&clusterConfig.WorkerNodes, flagWorkerNodes, flagWorkerNodesShort, 0, flagWorkerNodesHelp)
	cmd.Flags().StringVarP(&config.Providers.Libvirt.SessionURI, constants.FlagSshURI, constants.FlagSshURIShort, "", constants.FlagSshURIHelp)
	cmd.Flags().StringVarP(&config.Providers.Libvirt.SshKey, constants.FlagSshKey, constants.FlagSshKeyShort, "", constants.FlagSshKeyHelp)
	cmd.Flags().StringVarP(&config.BootVolumeContainerImage, constants.FlagBootVolumeContainerImage, constants.FlagBootVolumeContainerImageShort, "", constants.FlagBootVolumeContainerImageHelp)
	cmd.Flags().StringVarP(&clusterConfig.Name, constants.FlagClusterName, constants.FlagClusterNameShort, "", constants.FlagClusterNameHelp)
	cmd.Flags().StringVarP(&clusterConfig.Provider, constants.FlagProviderName, constants.FlagProviderNameShort, "", constants.FlagProviderNameHelp)
	cmd.Flags().StringVarP(&config.AutoStartUI, constants.FlagAutoStartUIName, constants.FlagAutoStartUINameShort, "", constants.FlagAutoStartUIHelp)
	cmd.Flags().StringVarP(&clusterConfig.KubeVersion, constants.FlagVersionName, constants.FlagVersionShort, "", constants.FlagKubernetesVersionHelp)
	cmd.Flags().StringVar(&clusterConfig.VirtualIp, flagVirtualIP, "", flagVirtualIPHelp)
	cmd.Flags().StringVar(&clusterConfig.LoadBalancer, flagLoadBalancer, "", flagLoadBalancerHelp)
	cmd.MarkFlagsMutuallyExclusive(flagVirtualIP, flagLoadBalancer)

	return cmd
}

// RunCmd runs the "ocne cluster start" command
func RunCmd(cmd *cobra.Command) error {
	c, cc, err := cmdutil.GetFullConfig(&config, &clusterConfig, clusterConfigPath)
	if err != nil {
		return err
	}
	if cc.Name == "" {
		cc.Name = "ocne"
	}
	if cc.Provider == "" {
		cc.Provider = pkgconst.ProviderTypeLibvirt
	}
	if cc.ControlPlaneNodes == 0 {
		cc.ControlPlaneNodes = 1
	}
	imageTransport := alltransports.TransportFromImageName(cc.BootVolumeContainerImage)
	if imageTransport == nil {
		// No transport protocol detected. Adding docker transport protocol as default.
		cc.BootVolumeContainerImage = "docker://" + cc.BootVolumeContainerImage
	}

	// Append the version to the boot volume container image registry path specified, if it does not already exist
	cc.BootVolumeContainerImage, err = cmdutil.EnsureBootImageVersion(cc.KubeVersion, cc.BootVolumeContainerImage)
	if err != nil {
		return err
	}

	// if the user has not overridden the osTag and the requested k8s version is not the default, make the osTag
	// match the k8s version
	if cc.OsTag == pkgconst.KubeVersion && cc.KubeVersion != pkgconst.KubeVersion {
		cc.OsTag = cc.KubeVersion
	}

	// if the provider is libvirt, update fields in the config otherwise the changes to the clusterConfig fields
	// will be overwritten
	if cc.Provider == pkgconst.ProviderTypeLibvirt {
		c.BootVolumeContainerImage = cc.BootVolumeContainerImage
		c.KubeVersion = cc.KubeVersion
		c.OsTag = cc.OsTag
	}

	_, _, err = start.Start(c, cc)

	return err
}
