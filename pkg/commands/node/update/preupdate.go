// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"
	"strings"

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
	"github.com/oracle-cne/ocne/pkg/util/script"
)

func processImage(registry string, tag string, img *v1.ContainerImage) (string, bool, bool) {
	haveCurrent := false
	exactMatch := false
	ret := ""

	imgName := fmt.Sprintf("%s:%s", registry, tag)
	imgPrefix := fmt.Sprintf("%s:", registry)
	currentName := fmt.Sprintf("%s:%s", registry, constants.CurrentTag)
	for _, name := range img.Names {
		if name == currentName {
			haveCurrent = true
			continue
		}
		if name == imgName {
			exactMatch = true
			ret = name
			continue
		}

		// If there is already an exact match, don't
		// look at this image
		if exactMatch {
			continue
		}
		if strings.HasPrefix(name, imgPrefix) {
			ret = name
		}
	}

	return ret, haveCurrent, exactMatch
}

func findTaggableImage(registry string, tag string, imgs []v1.ContainerImage) string {
	ret := ""
	foundExact := false
	for _, img := range imgs {
		imgName, haveCurrent, exactMatch := processImage(registry, tag, &img)

		// If there is a current image, don't tag something new
		if haveCurrent {
			log.Debugf("Have current image for %s", imgName)
			return ""
		}

		if exactMatch {
			ret = imgName
			foundExact = true
		} else if !foundExact {
			ret = imgName
		} else if ret == "" {
			ret = imgName
		}
	}
	return ret
}

func getKubeProxyTag(client kubernetes.Interface) (string, error) {
	ret := ""

	proxyDep, err := k8s.GetDaemonSet(client, constants.KubeProxyNamespace, constants.KubeProxyDaemonSet)
	if err != nil {
		return "", err
	}

	// If the kube-proxy image already makes sense, then do nothing.
	for _, c := range proxyDep.Spec.Template.Spec.Containers {
		imgInfo, err := image.SplitImage(c.Image)
		if err != nil {
			return "", err
		}

		if imgInfo.BaseImage != constants.KubeProxyImage {
			continue
		}

		log.Debugf("Found kube-proxy tag %s", imgInfo.Tag)
		ret = imgInfo.Tag
		break
	}

	return ret, err
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
		return nil
	}

	kubeProxyImg := findTaggableImage(constants.KubeProxyImage, kubeProxyTag, node.Status.Images)
	corednsImg := findTaggableImage(constants.CoreDNSImage, corednsTag, node.Status.Images)

	// If there is nothing to tag, then don't try.
	if kubeProxyImg  == "" && corednsImg == "" {
		return nil
	}

	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, namespace, []string{}, false)
	if err != nil {
		return err
	}

	// Build the script to run on the node
	tagScript := "#! /bin/bash"
	if kubeProxyImg != "" {
		log.Debugf("Tagging %s", kubeProxyImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(kubeProxyImg, constants.KubeProxyImage))
	}
	if corednsImg != "" {
		log.Debugf("Tagging %s", corednsImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(corednsImg, constants.CoreDNSImage))
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

func updateCoreDNS(client kubernetes.Interface) error {
	dnsDep, err := k8s.GetDeployment(client, constants.CoreDNSNamespace, constants.CoreDNSDeployment)
	if err != nil {
		return err
	}

	// If the kube-proxy image already makes sense, then do nothing.
	for i, c := range dnsDep.Spec.Template.Spec.Containers {
		imgInfo, err := image.SplitImage(c.Image)
		if err != nil {
			return err
		}


		if imgInfo.BaseImage != constants.CoreDNSImage {
			continue
		}

		// kube-proxy is already using current tag.  Nothing to do.
		if imgInfo.Tag == constants.CurrentTag {
			break
		}

		log.Debugf("Updating CoreDNS tag from %s to %s", imgInfo.Tag, constants.CurrentTag)
		dnsDep.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", constants.CoreDNSImage, constants.CurrentTag)

		// Once the daemonset is updated, the work is done.  Return
		_, err = k8s.UpdateDeployment(client, dnsDep, constants.CoreDNSNamespace)
		break
	}

	return err
}

func getCoreDNSTag(client kubernetes.Interface) (string, error) {
	ret := ""
	dnsDep, err := k8s.GetDeployment(client, constants.CoreDNSNamespace, constants.CoreDNSDeployment)
	if err != nil {
		return "", err
	}

	// If the kube-proxy image already makes sense, then do nothing.
	for _, c := range dnsDep.Spec.Template.Spec.Containers {
		imgInfo, err := image.SplitImage(c.Image)
		if err != nil {
			return "", err
		}

		if imgInfo.BaseImage != constants.CoreDNSImage {
			continue
		}

		log.Debugf("Found CoreDNS tag %s", imgInfo.Tag)
		ret = imgInfo.Tag
		break
	}

	return ret, err
}

func preUpdate(restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, nodes *v1.NodeList) error {
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

	err = updateCoreDNS(client)
	if err != nil {
		return nil
	}
	return nil
}
