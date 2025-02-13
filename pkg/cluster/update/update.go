// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/script"
)

func getDaemonSetTag(client kubernetes.Interface, dsNamespace string, dsName string, registry string) (string, error) {
	ret := ""

	dsDep, err := k8s.GetDaemonSet(client, dsNamespace, dsName)
	if err != nil {
		return "", err
	}

	// If the kube-proxy image already makes sense, then do nothing.
	for _, c := range dsDep.Spec.Template.Spec.Containers {
		imgInfo, err := image.SplitImage(c.Image)
		if err != nil {
			return "", err
		}

		if imgInfo.BaseImage != registry {
			continue
		}

		log.Debugf("Found %s tag %s", registry, imgInfo.Tag)
		ret = imgInfo.Tag
		break
	}

	return ret, err
}

func getDeploymentTag(client kubernetes.Interface, depNamespace string, depName string, registry string) (string, error) {
	ret := ""
	dep, err := k8s.GetDeployment(client, depNamespace, depName)
	if err != nil {
		return "", err
	}

	// If the kube-proxy image already makes sense, then do nothing.
	for _, c := range dep.Spec.Template.Spec.Containers {
		imgInfo, err := image.SplitImage(c.Image)
		if err != nil {
			return "", err
		}

		if imgInfo.BaseImage != registry {
			continue
		}

		log.Debugf("Found %s tag %s", registry, imgInfo.Tag)
		ret = imgInfo.Tag
		break
	}

	return ret, err
}

func getKubeProxyTag(client kubernetes.Interface) (string, error) {
	return getDaemonSetTag(client, constants.KubeProxyNamespace, constants.KubeProxyDaemonSet, constants.KubeProxyImage)
}

func tagCommand(imgName string, registry string) string {
	return fmt.Sprintf("chroot /hostroot podman tag %s %s:%s", imgName, registry, constants.CurrentTag)
}

func tagOnNode(node *v1.Node, restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string) error {
	namespace := constants.OCNESystemNamespace

	log.Debugf("Finding images to tag on %s", node.ObjectMeta.Name)

	kubeProxyTag, err := getKubeProxyTag(client)
	if err != nil {
		return err
	}
	corednsTag, err := getCoreDNSTag(client)
	if err != nil {
		return err
	}
	flannelTag, err := getDaemonSetTag(client, constants.CNIFlannelNamespace, constants.CNIFlannelDaemonSet, constants.CNIFlannelImage)
	if err != nil {
		return err
	}
	uiTag, err := getDeploymentTag(client, constants.UINamespace, constants.UIDeployment, constants.UIImage)
	if err != nil {
		return err
	}

	kubeProxyImg, kubeProxyCurrent, _ := k8s.GetImageCandidate(constants.KubeProxyImage, constants.CurrentTag, kubeProxyTag, node)
	corednsImg, corednsCurrent, _ := k8s.GetImageCandidate(constants.CoreDNSImage, constants.CurrentTag, corednsTag, node)
	flannelImg, flannelCurrent, _ := k8s.GetImageCandidate(constants.CNIFlannelImage, constants.CurrentTag, flannelTag, node)
	uiImg, uiCurrent, _ := k8s.GetImageCandidate(constants.UIImage, constants.CurrentTag, uiTag, node)

	// If there is nothing to tag, then don't try.
	if kubeProxyCurrent && corednsCurrent && flannelCurrent && uiCurrent {
		return nil
	}

	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, namespace, []string{}, false)
	if err != nil {
		return err
	}

	// Build the script to run on the node
	tagScript := "#! /bin/bash"
	if !kubeProxyCurrent {
		log.Debugf("Tagging %s", kubeProxyImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(kubeProxyImg, constants.KubeProxyImage))
	}
	if !corednsCurrent {
		log.Debugf("Tagging %s", corednsImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(corednsImg, constants.CoreDNSImage))
	}
	if !flannelCurrent {
		log.Debugf("Tagging %s", flannelImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(flannelImg, constants.CNIFlannelImage))
	}
	if !uiCurrent {
		log.Debugf("Tagging %s", uiImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(uiImg, constants.UIImage))
	}

	return script.RunScript(client, kcConfig, node.ObjectMeta.Name, namespace, "tag-images", tagScript, []v1.EnvVar{})
}

func updateKubeProxy(client kubernetes.Interface, kubeConfigPath string) error {
	// If kube-proxy is already installed as an application, don't try
	// to install it again.
	exists, err := install.DoesReleaseExist(constants.KubeProxyRelease, kubeConfigPath, constants.KubeProxyNamespace)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("kube-proxy application already exists")
		return nil
	}

	// Calculating the correct overrides based solely on the kubeconfig is
	// hard, and is not tolerant to user customizations.  It's much easier
	// to simply use the values that are already there.
	cm, err := k8s.GetConfigmap(client, constants.KubeProxyNamespace, constants.KubeProxyConfigMap)
	if err != nil {
		return err
	}

	// Of course, that configuration may be corrupt.  Make sure it's not.
	conf, ok := cm.Data[constants.KubeProxyConfigMapConfig]
	if !ok {
		return fmt.Errorf("ConfigMap %s in %s did not have a %s key", constants.KubeProxyConfigMap, constants.KubeProxyNamespace, constants.KubeProxyConfigMapConfig)
	}

	confParsed := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(conf), confParsed)
	if err != nil {
		return err
	}

	kcfg, ok := cm.Data[constants.KubeProxyConfigMapKubeconfig]
	if !ok {
		return fmt.Errorf("ConfigMap %s in %s did not have a %s key", constants.KubeProxyConfigMap, constants.KubeProxyNamespace, constants.KubeProxyConfigMapKubeconfig)
	}

	kcfgParsed := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(kcfg), kcfgParsed)
	if err != nil {
		return err
	}

	return install.InstallApplications([]install.ApplicationDescription{
		install.ApplicationDescription{
			Force: true,
			Application: &types.Application{
				Name:      constants.KubeProxyChart,
				Namespace: constants.KubeProxyNamespace,
				Release:   constants.KubeProxyRelease,
				Version:   constants.KubeProxyVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"kubeconfig": kcfgParsed,
					"config": confParsed,
				},
			},
		},
	}, kubeConfigPath, false)
}

func updateCoreDNS(kubeConfigPath string) error {
	exists, err := install.DoesReleaseExist(constants.CoreDNSRelease, kubeConfigPath, constants.CoreDNSNamespace)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("CoreDNS application already exists")
		return nil
	}

	return install.InstallApplications([]install.ApplicationDescription{
		install.ApplicationDescription{
			Force: true,
			Application: &types.Application{
				Name: constants.CoreDNSChart,
				Namespace: constants.CoreDNSNamespace,
				Release: constants.CoreDNSRelease,
				Version: constants.CoreDNSVersion,
				Catalog: catalog.InternalCatalog,
			},
		},
	}, kubeConfigPath, false)
}

func getCoreDNSTag(client kubernetes.Interface) (string, error) {
	return getDeploymentTag(client, constants.CoreDNSNamespace, constants.CoreDNSDeployment, constants.CoreDNSImage)
}

func oneThirtyAndLower(restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, nodes *v1.NodeList) error {
	// It's not possible to get past k8s 1.30 and still have to
	// do this.
	doIt := false
	for _, n := range nodes.Items {
		res, err := util.CompareVersions(n.Status.NodeInfo.KubeletVersion, "1.30")
		if err != nil {
			return err
		}

		if res < 1 {
			doIt = true
			break
		}
	}

	if !doIt {
		log.Debugf("Skipping updates that only apply to Kubernetes versions 1.30 and lower")
		return nil
	}

	// Check for presence of "current" tags for kube-proxy
	// and coredns.  Nodes that don't have them, need them.
	haveError := false
	haveSuccess := false
	for _, node := range nodes.Items {
		err := tagOnNode(&node, restConfig, client, kubeConfigPath)
		if err != nil {
			haveError = true
			log.Errorf("Could not set image tags on %s: %v", node.ObjectMeta.Name, err)
		} else {
			haveSuccess = true
		}
	}

	if !haveSuccess && haveError {
		return fmt.Errorf("Could not tag images on any nodes")
	}

	// Once at least some nodes have the current tags, update the
	// kube-proxy daemonset and coredns deployment to use them.
	err := updateKubeProxy(client, kubeConfigPath)
	if err != nil {
		return err
	}

	err = updateCoreDNS(kubeConfigPath)
	if err != nil {
		return nil
	}
	return nil
}

// updateFuncs is an ordered list of update functions to run.
var updateFuncs = []func(*rest.Config, kubernetes.Interface, string, *v1.NodeList)error{
	oneThirtyAndLower,
}
// Update applies the cumulative set of changes that have built
// up over time as configuration deficiences have been discovered
// and repaired.
func Update(restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, nodes *v1.NodeList) error {
	for _, f := range updateFuncs {
		err := f(restConfig, client, kubeConfigPath, nodes)
		if err != nil {
			return err
		}
	}
	return nil
}
