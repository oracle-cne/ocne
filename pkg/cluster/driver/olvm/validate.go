// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/config/types"
)

// validateConfig - validateConfig the configuration. These validations may be moved to a
// validating webhook in the future.
func validateConfig(clusterConfig *types.ClusterConfig) error {
	// Unlike other cluster drivers, it is not possible to have zero worker nodes.
	// Cluster API will not create control plane nodes with taints removed,
	// and it can get upset if they are removed.
	// Require at least one.
	//
	// If someone really wants to have no workers, then they are free
	// to pass in a cluster definition.
	if clusterConfig.WorkerNodes == 0 {
		clusterConfig.WorkerNodes = 1
	}

	// It's also not possible to have zero control plane nodes.
	if clusterConfig.ControlPlaneNodes == 0 {
		clusterConfig.ControlPlaneNodes = 1
	}

	provider := clusterConfig.Providers.Olvm
	msgFormat := "The configuration parameter %s is required"

	if len(provider.DatacenterName) == 0 {
		return fmt.Errorf(msgFormat, "providers.olvm.olvmDatacenterName")
	}
	if len(provider.OlvmAPIServer.ServerURL) == 0 {
		return fmt.Errorf(msgFormat, "providers.olvm.olvmOvirtAPIServer.serverURL")
	}
	if err := validateMachine(provider.ControlPlaneMachine, msgFormat, "providers.olvm.controlPlaneMachine"); err != nil {
		return err
	}
	if err := validateMachine(provider.WorkerMachine, msgFormat, "providers.olvm.workerMachine"); err != nil {
		return err
	}
	return nil
}

// validateMachine - validate a machine configuration
func validateMachine(machine types.OlvmMachine, fmtString string, basePath string) error {
	if len(machine.OlvmOvirtClusterName) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "olvmOvirtClusterName"))
	}
	if len(machine.VMTemplateName) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "vmTemplateName"))
	}
	if err := validateNetwork(machine.OlvmNetwork, fmtString, fmt.Sprintf("%s.%s", basePath, "olvmNetwork")); err != nil {
		return err
	}
	if err := validateVirtualMachine(machine.VirtualMachine, fmtString, fmt.Sprintf("%s.%s", basePath, "virtualMachine")); err != nil {
		return err
	}
	return nil
}

// validateNetwork - validate a network configuration
func validateNetwork(network types.OlvmNetwork, fmtString string, basePath string) error {
	if len(network.NetworkName) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "networkName"))
	}
	if len(network.VnicProfileName) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "vnicProfileName"))
	}
	return nil
}

// validateVirtualMachine - validate a virtual machine configuration
func validateVirtualMachine(vm types.OlvmVirtualMachine, fmtString string, basePath string) error {
	if err := validateVirtualMachineNetwork(vm.Network, fmtString, fmt.Sprintf("%s.%s", basePath, "network")); err != nil {
		return err
	}
	return nil
}

func validateVirtualMachineNetwork(vmn types.OlvmVirtualMachineNetwork, fmtString string, basePath string) error {
	if err := validateIpV4(vmn.IPV4, fmtString, fmt.Sprintf("%s.%s", basePath, "ipv4")); err != nil {
		return err
	}
	return nil
}

// validateIpV4 - validate IPV4 configuration
func validateIpV4(ipv4 types.OlvmIPV4, fmtString string, basePath string) error {
	if len(ipv4.IpAddresses) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "ipAddresses"))
	}
	if len(ipv4.Subnet) == 0 {
		return fmt.Errorf(fmtString, fmt.Sprintf("%s.%s", basePath, "subnet"))
	}
	return nil
}
