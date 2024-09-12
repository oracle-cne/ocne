// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package show

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

// Show takes in a set of options and returns the appropriate output about the values of a specific release
func Show(opt application.ShowOptions) (string, error) {
	var finalOutput string
	kubeInfo, err := client.CreateKubeInfo(opt.KubeConfigPath)
	if err != nil {
		return finalOutput, err
	}
	if opt.Namespace == "" {
		opt.Namespace, err = client.GetNamespaceFromConfig(opt.KubeConfigPath)
		if err != nil {
			return finalOutput, err
		}
	}
	if opt.Difference {
		finalOutput, err = createDifferenceMessage(kubeInfo, opt.ReleaseName, opt.Namespace)
		if err != nil {
			return finalOutput, err
		}
	} else if opt.Computed {
		finalOutput, err = createGeneratedMessage(kubeInfo, opt.ReleaseName, opt.Namespace)
		if err != nil {
			return finalOutput, err
		}
	} else {
		finalOutput, err = createOverrideMessage(kubeInfo, opt.ReleaseName, opt.Namespace)
		if err != nil {
			return finalOutput, err
		}
	}
	metadataOutput, err := createMetadataOutput(kubeInfo, opt.ReleaseName, opt.Namespace)
	if err != nil {
		return finalOutput, err
	}
	finalOutput = finalOutput + metadataOutput

	return finalOutput, nil
}

// createOverrideMessage returns the output of the 'helm get values' command
func createOverrideMessage(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	commandOutput, err := helm.GetValues(kubeInfo, releaseName, namespace)
	if err != nil {
		return "", err
	}
	return string(commandOutput), nil
}

// createGeneratedMessage returns the output of the 'helm get values -a' command
func createGeneratedMessage(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	commandOutput, err := helm.GetAllValues(kubeInfo, releaseName, namespace)
	if err != nil {
		return "", err
	}
	return string(commandOutput), nil
}

// createDifference message combines the output of the 'helm get values' command
// and the 'helm get values -a' command and returns the two outputs
func createDifferenceMessage(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	finalOutput := ""
	overrideCommandOutput, err := createOverrideMessage(kubeInfo, releaseName, namespace)
	if err != nil {
		return "", err
	}
	finalOutput = "OVERRIDE/USER-SUPPLIED VALUES:\n" + overrideCommandOutput + "----------------\n"
	generatedCommandOutput, err := createGeneratedMessage(kubeInfo, releaseName, namespace)
	if err != nil {
		return "", err
	}
	finalOutput = finalOutput + ("GENERATED VALUES:\n") + generatedCommandOutput
	return finalOutput, nil
}

// createMetadataOutput creates the metadata output to be generated with the application show command
// It is the same information for a chart as seen in the application ls command
func createMetadataOutput(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	var metadataOutput string
	metadataForRelease, err := helm.GetMetadata(kubeInfo, releaseName, namespace)
	if err != nil {
		return metadataOutput, err
	}
	metadataOutput = "----------------\n" + fmt.Sprintf("NAME: %s\n", metadataForRelease.Name)
	metadataOutput = metadataOutput + fmt.Sprintf("NAMESPACE: %s\n", metadataForRelease.Namespace)
	metadataOutput = metadataOutput + fmt.Sprintf("CHART: %s\n", metadataForRelease.Chart)
	metadataOutput = metadataOutput + fmt.Sprintf("STATUS: %s\n", metadataForRelease.Status)
	metadataOutput = metadataOutput + fmt.Sprintf("REVISION: %d\n", metadataForRelease.Revision)
	metadataOutput = metadataOutput + fmt.Sprintf("APPVERSION: %s\n", metadataForRelease.AppVersion)
	return metadataOutput, nil
}
