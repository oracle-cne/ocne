// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stage

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/script"
)

const (
	envNodeName = "NODE_NAME"
	namespace   = constants.OCNESystemNamespace
)

// StageOptions are the options for the staget command
type StageOptions struct {
	// KubeConfigPath is the path to the optional kubeconfig file
	KubeConfigPath string

	// KubeVersion is the version of Kubernetes to update to
	KubeVersion string

	// OsRegistry is the name of the osRegistry that will be stored in the update.yaml for all nodes during a stage
	OsRegistry string

	// Transport is the type of transport that will be stored in the update.yaml for all nodes during a stage
	Transport string

	// Timeout is how long the CLI waits for a pod to become available on a node
	Timeout string

	// ClusterConfig can be used to get configuration values from a cluster config
	ClusterConfig *types.ClusterConfig
	Config *types.Config
}

// Stage stages a cluster update
func Stage(o StageOptions) error {
	//check that the user is supplying a major.minor version
	match, err := regexp.MatchString("^[1-9]\\.[0-9][0-9]$", o.KubeVersion)
	if !match {
		return errors.New("version in major.minor format is not provided. Please choose a version of Kubernetes in the form of major.minor, such as 1.30")
	}
	// check that it is a major/minor version that we support
	if _, err := versions.GetKubernetesVersions(o.KubeVersion); err != nil {
		return errors.New("the kubernetes version is unsupported. Please choose a supported version of Kubernetes to update to")
	}

	// If a cluster config is given, do any provider-specific staging
	if o.ClusterConfig != nil {
		cd, err := driver.CreateDriver(o.Config, o.ClusterConfig)
		if err != nil {
			return err
		}

		err = cd.Stage(o.KubeVersion)
		if err != nil {
			return err
		}
	}
	return nil

	// get a kubernetes client
	restConfig, KClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}

	// ensure the Namespace exists
	k8s.CreateNamespaceIfNotExists(KClient, namespace)

	// get config needed to use kubectl
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, o.KubeConfigPath, namespace, nil, false)
	if err != nil {
		return err
	}

	nodeList, err := k8s.GetNodeList(KClient)
	if err != nil {
		return err
	}

	listOfMinorVersions, err := stageNodeClusterCheck(nodeList)
	if err != nil {
		return err
	}
	err = stageNodeVersionCheck(listOfMinorVersions, o.KubeVersion)
	if err != nil {
		return err
	}

	stageCount := 0
	for _, node := range nodeList.Items {
		if k8s.IsNodeReady(node.Status) {
			err := StageNode(KClient, kcConfig, o, node.Name, node.Status.NodeInfo.KubeletVersion)
			if err != nil {
				log.Errorf("Error staging node %s: %v", node.Name, err)
			} else {
				stageCount += 1
			}
		} else {
			log.Infof("Skipping node %s because it is not ready", node.Name)
		}
	}
	if stageCount == 0 {
		return fmt.Errorf("Unable to stage any nodes")
	}

	_, _, err = util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		// Update config map to change version - kubeadmconfig
		configmap, err := k8s.GetConfigmap(KClient, constants.KubeNamespace, constants.KubeCMName)
		if err != nil {
			return nil, false, err
		}
		data := configmap.Data["ClusterConfiguration"]
		compile, err := regexp.Compile("kubernetesVersion:.*[$\\n]")
		if err != nil {
			return nil, false, err
		}
		newData := compile.ReplaceAll([]byte(data), []byte("kubernetesVersion: v"+o.KubeVersion+".0"+"\n"))
		configmap.Data["ClusterConfiguration"] = string(newData)
		_, err = k8s.UpdateConfigMap(KClient, configmap, constants.KubeNamespace)
		if err != nil {
			return nil, false, err
		}

		return nil, true, nil
	}, nil, 60*time.Second)

	return err
}

// StageNode stages a cluster update
// 1. Changes the tag in update.yaml to the new kubernetes version
// 2. restarts update.service
func StageNode(client kubernetes.Interface, kcConfig *kubectl.KubectlConfig, o StageOptions, nodename string, currentNodeVersion string) error {
	log.Info("Running node stage")
	_, _, err := util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		err := script.RunScript(client, kcConfig, nodename, namespace, "stage-node", stageNodeScript, []corev1.EnvVar{
			{Name: envNodeName, Value: nodename},
			{Name: "NEW_K8S_VERSION", Value: o.KubeVersion},
			{Name: "NEW_REGISTRY", Value: o.OsRegistry},
			{Name: "NEW_TRANSPORT", Value: o.Transport},
		})
		return nil, false, err
	}, nil, 60*time.Second)
	if err != nil {
		return err
	}

	log.Infof("Node %s successfully staged", nodename)
	return nil
}

// stageNodeClusterCheck checks that the cluster is in appropriate condition by checking that all nodes within the cluster are within one minor version of each other
func stageNodeClusterCheck(nodeList *corev1.NodeList) ([]string, error) {
	listOfVersions := []*semver.Version{}
	listOfMinorVersions := []string{}
	minorVersionMap := map[string]bool{}
	for _, node := range nodeList.Items {
		version, err := semver.NewVersion(node.Status.NodeInfo.KubeletVersion)
		if err != nil {
			return listOfMinorVersions, err
		}
		listOfVersions = append(listOfVersions, version)
		minorVersionMap[extractMajorAndMinorVersionFromVersion(version)] = true
	}
	for k, _ := range minorVersionMap {
		listOfMinorVersions = append(listOfMinorVersions, k)
	}
	// This block of code checks that the cluster's nodes are either one version or are within one minor version of each-other (1.26 and 1.27, for example)
	sort.Sort(semver.Collection(listOfVersions))
	lowestVersion := listOfVersions[0]
	increasedLowerVersion := lowestVersion.IncMinor()
	highestVersion := listOfVersions[len(listOfVersions)-1]
	equalMajorAndMinor, err := semver.NewConstraint("=" + extractMajorAndMinorVersionFromVersion(lowestVersion) + ".x")
	if err != nil {
		return listOfMinorVersions, fmt.Errorf("error when extracting major and minor version from version string in Kubernetes cluster")
	}
	equalMajorMinorIncreased, err := semver.NewConstraint("=" + extractMajorAndMinorVersionFromVersion(&increasedLowerVersion) + ".x")
	if err != nil {
		return listOfMinorVersions, fmt.Errorf("error when extracting major and minor version from version string in Kubernetes cluster")
	}
	if equalMajorAndMinor.Check(highestVersion) || equalMajorMinorIncreased.Check(highestVersion) {
		return listOfMinorVersions, nil
	}

	return listOfMinorVersions, fmt.Errorf("the cluster's nodes don't share the same version or the cluster's nodes aren't within a minor version of each other ")
}

// stageNodeVersionCheck checks that the version that the user is staging to is acceptable
// When the cluster's nodes are all running the same minor version, the user's version has to be that same minor version or exactly one minor version above it
// For example, if the cluster's nodes are all running 1.28, the user's version to stage must be 1.28 or 1.29
// In the case, where there are 2 Kubernetes versions being used, and they are one minor version apart from each other
// The user must stage the greater minor version
// For example, if the cluster has nodes that are running 1.27 and 1.28, the user must stage 1.28

func stageNodeVersionCheck(listOfMinorVersions []string, versionToStage string) error {
	versionToStageSemVer, err := semver.NewVersion(versionToStage)
	if err != nil {
		return err
	}
	firstMajorMinor, err := semver.NewVersion(listOfMinorVersions[0])
	if err != nil {
		return err
	}
	if len(listOfMinorVersions) == 1 {
		firstMajorMinorIncreased := firstMajorMinor.IncMinor()
		equalMajorMinorIncreased, err := semver.NewConstraint("=" + extractMajorAndMinorVersionFromVersion(&firstMajorMinorIncreased))
		if err != nil {
			return err
		}
		equalMajorMinorSame, err := semver.NewConstraint("=" + extractMajorAndMinorVersionFromVersion(firstMajorMinor))
		if err != nil {
			return err
		}
		if equalMajorMinorIncreased.Check(versionToStageSemVer) || equalMajorMinorSame.Check(versionToStageSemVer) {
			return nil
		} else {
			return errors.New(fmt.Sprintf("the minor version given to the stage command is not exactly the same minor version or one minor version higher than the version currently being used by nodes in the cluster. The version to stage should be %s or %s", extractMajorAndMinorVersionFromVersion(firstMajorMinor), extractMajorAndMinorVersionFromVersion(&firstMajorMinorIncreased)))
		}
	}
	secondMajorMinor, err := semver.NewVersion(listOfMinorVersions[1])
	if err != nil {
		return err
	}
	var largerMajorMinor *semver.Version
	if firstMajorMinor.GreaterThan(secondMajorMinor) {
		largerMajorMinor = firstMajorMinor
	} else {
		largerMajorMinor = secondMajorMinor
	}
	equalToLargerMajorAndMinor, err := semver.NewConstraint("=" + extractMajorAndMinorVersionFromVersion(largerMajorMinor))
	if err != nil {
		return err
	}
	if equalToLargerMajorAndMinor.Check(versionToStageSemVer) {
		return nil
	}
	return errors.New(fmt.Sprintf("the minor version given to stage is not the larger of the two minor versions currently being used by nodes in the cluster. The version to stage should be %s", extractMajorAndMinorVersionFromVersion(largerMajorMinor)))
}

// extractMajorAndMinorVersionFromVersions converts a semver Version to a major.minor string
func extractMajorAndMinorVersionFromVersion(version *semver.Version) string {
	return fmt.Sprintf("%d.%d", version.Major(), version.Minor())
}
