// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/release"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/application/ls"
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
		if strings.Contains(err.Error(), "not found") {
			return "", nil
		}
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
		if strings.Contains(err.Error(), "not found") {
			return "", nil
		}
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

func tagOnNode(node *v1.Node, restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, kubeProxyTag string, corednsTag string, flannelTag string, uiTag string) error {
	namespace := constants.OCNESystemNamespace

	log.Debugf("Finding images to tag on %s", node.ObjectMeta.Name)


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
	if !kubeProxyCurrent && kubeProxyTag != "" && kubeProxyImg != "" {
		log.Debugf("Tagging %s", kubeProxyImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(kubeProxyImg, constants.KubeProxyImage))
	}
	if !corednsCurrent && corednsTag != "" && corednsImg != "" {
		log.Debugf("Tagging %s", corednsImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(corednsImg, constants.CoreDNSImage))
	}
	if !flannelCurrent && flannelTag != "" && flannelImg != "" {
		log.Debugf("Tagging %s", flannelImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(flannelImg, constants.CNIFlannelImage))
	}
	if !uiCurrent && uiTag != "" && uiImg != "" {
		log.Debugf("Tagging %s", uiImg)
		tagScript = fmt.Sprintf("%s\n%s", tagScript, tagCommand(uiImg, constants.UIImage))
	}

	return script.RunScript(client, kcConfig, node.ObjectMeta.Name, namespace, "tag-images", tagScript, []v1.EnvVar{})
}

func getRelease(release string, namespace string, kubeConfigPath string) (*release.Release, error) {
	releases, err := ls.List(application.LsOptions{
		KubeConfigPath: kubeConfigPath,
		Namespace: namespace,
		All: false,
	})
	if err != nil {
		return nil, err
	}

	for _, rel := range releases {
		if rel.Name == release {
			if rel.Config == nil {
				rel.Config = map[string]interface{}{}
			}
			return rel, nil
		}
	}

	return nil, nil
}


func updateKubeProxy(client kubernetes.Interface, kubeConfigPath string) error {
	// If kube-proxy is already installed as an application, don't try
	// to install it again.
	proxyRelease, err := getRelease(constants.KubeProxyRelease, constants.KubeProxyNamespace, kubeConfigPath)

	// If kube-proxy is not installed at all, don't force it
	_, err = k8s.GetDaemonSet(client, constants.KubeProxyNamespace, constants.KubeProxyDaemonSet)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}

		return err
	}

	// If the release was found, just update the tag.  That way the complex
	// calculation of the configuration is avoided.
	if proxyRelease != nil {
		// If the tag is already 'current', assume this is already correctly
		// configured and return.
		tag, found, err := unstructured.NestedString(proxyRelease.Config, "image", "tag")
		if err != nil {
			return err
		} else if !found || tag == constants.KubeProxyTag {
			// If the 'current' tag is already assigned, don't do anything.
			return nil
		}

		err = unstructured.SetNestedField(proxyRelease.Config, constants.KubeProxyTag, "image", "tag")
		if err != nil {
			return err
		}
		return install.UpdateApplications([]install.ApplicationDescription{
			install.ApplicationDescription{
				Force: false,
				Application: &types.Application{
					Name:      constants.KubeProxyChart,
					Namespace: constants.KubeProxyNamespace,
					Release:   constants.KubeProxyRelease,
					Version:   constants.KubeProxyVersion,
					Catalog:   catalog.InternalCatalog,
					Config:    proxyRelease.Config,
					},
				},
		}, kubeConfigPath, false)
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
				Config:    map[string]interface{}{
						"image": map[string]interface{}{
							"tag": constants.KubeProxyTag,
						},
						"kubeconfig": kcfgParsed,
						"config": confParsed,
					},
			},
		},
	}, kubeConfigPath, false)
}

func updateCoreDNS(client kubernetes.Interface, kubeConfigPath string) error {
	// If CoreDNS is already installed, don't do it again
	corednsRelease, err := getRelease(constants.CoreDNSRelease, constants.CoreDNSNamespace, kubeConfigPath)
	if err != nil {
		return err
	}

	// If CoreDNS is not installed at all, don't for it
	_, err = k8s.GetDeployment(client, constants.CoreDNSNamespace, constants.CoreDNSDeployment)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}

		return err
	}


	if corednsRelease != nil {
		tag, found, err := unstructured.NestedString(corednsRelease.Config, "image", "tag")
		if err != nil {
			return err
		} else if !found || tag == constants.CoreDNSTag {
			// If the 'current' tag is already assigned, don't do anything.
			return nil
		}

		err = unstructured.SetNestedField(corednsRelease.Config, constants.CoreDNSTag, "image", "tag")
		if err != nil {
			return err
		}

		return install.UpdateApplications([]install.ApplicationDescription{
			install.ApplicationDescription{
				Force: true,
				Application: &types.Application{
					Name:      constants.CoreDNSChart,
					Namespace: constants.CoreDNSNamespace,
					Release:   constants.CoreDNSRelease,
					Version:   constants.CoreDNSVersion,
					Catalog:   catalog.InternalCatalog,
					Config:    corednsRelease.Config,
				},
			},
		}, kubeConfigPath, false)
	}

	kubeletConfig, err := k8s.GetKubeletConfig(client)
	if err != nil {
		return err
	}

	if len(kubeletConfig.ClusterDNS) == 0 {
		return fmt.Errorf("cluster does not have a DNS service ip")
	}

	return install.InstallApplications([]install.ApplicationDescription{
		install.ApplicationDescription{
			Force: true,
			Application: &types.Application{
				Name:      constants.CoreDNSChart,
				Namespace: constants.CoreDNSNamespace,
				Release:   constants.CoreDNSRelease,
				Version:   constants.CoreDNSVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    map[string]interface{}{
					"image": map[string]interface{}{
						"tag": constants.CoreDNSTag,
					},
					"service": map[string]interface{}{
						"clusterIP": kubeletConfig.ClusterDNS[0],
						"clusterIPs": kubeletConfig.ClusterDNS,
					},
				},
			},
		},
	}, kubeConfigPath, false)
}

func getCoreDNSTag(client kubernetes.Interface) (string, error) {
	return getDeploymentTag(client, constants.CoreDNSNamespace, constants.CoreDNSDeployment, constants.CoreDNSImage)
}

func updateFlannel(kubeConfigPath string) error {
	flannelRelease, err := getRelease(constants.CNIFlannelRelease, constants.CNIFlannelNamespace, kubeConfigPath)
	if err != nil {
		return err
	}

	// Don't force install of Flannel if it isn't installed.
	if flannelRelease == nil {
		log.Debugf("Flannel application is not installed")
		return nil
	}

	// If the tag is already correct, leave it alone
	tag, found, err := unstructured.NestedString(flannelRelease.Config, "flannel", "image", "tag")
	if err != nil {
		return err
	} else if !found || tag == constants.CNIFlannelImageTag {
		return nil
	}

	err = unstructured.SetNestedField(flannelRelease.Config, constants.CNIFlannelImageTag, "flannel", "image", "tag")
	if err != nil {
		return err
	}

	return install.UpdateApplications([]install.ApplicationDescription{
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CNIFlannelChart,
				Namespace: constants.CNIFlannelNamespace,
				Release:   constants.CNIFlannelRelease,
				Version:   constants.CNIFlannelVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    flannelRelease.Config,
			},
		},
	}, kubeConfigPath, false)
}

func updateUI(kubeConfigPath string) error {
	uiRelease, err := getRelease(constants.UIRelease, constants.UINamespace, kubeConfigPath)
	if err != nil {
		return err
	}

	// Don't force install of the UI if it isn't installed
	if uiRelease == nil {
		log.Debugf("UI application is not installed")
		return nil
	}

	// If the tag is already correct, leave it alone
	tag, _, err := unstructured.NestedString(uiRelease.Config, "image", "tag")
	if err != nil {
		return err
	} else if tag == constants.UIImageTag {
		return nil
	}

	err = unstructured.SetNestedField(uiRelease.Config, constants.UIImageTag, "image", "tag")
	if err != nil {
		return err
	}

	return install.UpdateApplications([]install.ApplicationDescription{
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.UIChart,
				Namespace: constants.UINamespace,
				Release:   constants.UIRelease,
				Version:   constants.UIVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    uiRelease.Config,
			},
		},
	}, kubeConfigPath, false)
}

func oneThirtyAndLower(restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, nodes *v1.NodeList) error {
	// It's not possible to get past k8s 1.30 and still have to
	// do this.
	doIt := false
	for _, n := range nodes.Items {
		res, err := util.CompareVersions(n.Status.NodeInfo.KubeletVersion, "1.31")
		if err != nil {
			return err
		}

		if res == -1 {
			doIt = true
			break
		}
	}

	if !doIt {
		log.Debugf("Skipping updates that only apply to Kubernetes versions 1.30 and lower")
		return nil
	}

	// Get tags for any images that may need to be updated.
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

	// Check for presence of "current" tags for kube-proxy
	// and coredns.  Nodes that don't have them, need them.
	haveError := false
	haveSuccess := false
	for _, node := range nodes.Items {
		err := tagOnNode(&node, restConfig, client, kubeConfigPath, kubeProxyTag, corednsTag, flannelTag, uiTag)
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
	err = updateKubeProxy(client, kubeConfigPath)
	if err != nil {
		return err
	}

	err = updateCoreDNS(client, kubeConfigPath)
	if err != nil {
		return nil
	}

	err = updateFlannel(kubeConfigPath)
	if err != nil {
		return nil
	}

	err = updateUI(kubeConfigPath)
	if err != nil {
		return nil
	}
	return nil
}

var vipableProviderIds = []string{
	"olvm://",
}

func isVipableProvider(provider string) bool {
	if provider == "" {
		return true
	}

	for _, vp := range vipableProviderIds {
		if strings.HasPrefix(provider, vp) {
			return true
		}
	}
	return false
}

func unitToPath(u string) string {
	return fmt.Sprintf("/etc/systemd/system/%s", u)
}

func generateVipUpdateScript(bindPort uint16, altPort uint16, virtualIp string) (string, error) {
	keepalivedCheckScript, err := ignition.GenerateKeepalivedCheckScript(bindPort, altPort, virtualIp)
	if err != nil {
		return "", err
	}

	in := struct {
		Units []string
		UnitsToEnable []string
		UnitsToStart []string
		Files map[string]string
	}{
		Units: []string{
			ignition.KeepalivedRefreshServiceName,
			ignition.NginxRefreshServiceName,
		},
		UnitsToEnable: []string{
			ignition.NginxRefreshPathName,
			ignition.KeepalivedRefreshPathName,
			ignition.KeepalivedRefreshServiceName,
			ignition.NginxRefreshServiceName,

		},
		UnitsToStart: []string {
			ignition.NginxRefreshPathName,
			ignition.KeepalivedRefreshPathName,
		},
		Files: map[string]string{
			unitToPath(ignition.KeepalivedRefreshServiceName): ignition.GetKeepalivedRefreshUnit(),
			unitToPath(ignition.KeepalivedRefreshPathName): ignition.GetKeepalivedRefreshPathUnit(),
			unitToPath(ignition.NginxRefreshServiceName): ignition.GetNginxRefreshUnit(),
			unitToPath(ignition.NginxRefreshPathName): ignition.GetNginxRefreshPathUnit(),
			unitToPath(ignition.NginxServiceName): ignition.NginxService,
			ignition.KeepAlivedCheckScriptPath: keepalivedCheckScript,
		},
	}

	// Base64 encode the file contents to avoid having
	// to deal with escaping and whatnot.
	for p, c := range in.Files {
		in.Files[p] = base64.StdEncoding.EncodeToString([]byte(c))
	}

	ret, err := util.TemplateToString(UpdateVipConfiguration, &in)
	if err != nil {
		return "", err
	}

	return ret, nil
}

func parseEndpoint(endpoint string) (string, uint16, error) {
	endpointHost, endpointPortStr, err := kubeadmutil.ParseHostPort(endpoint)
	if err != nil {
		return "", 0, err
	}

	endpointPort64, err := strconv.ParseUint(endpointPortStr, 10, 16)
	if err != nil {
		return "", 0, err
	}

	return endpointHost, uint16(endpointPort64), nil
}

func getEndpointForNode(node *v1.Node, pods *v1.PodList) (string, uint16, error) {
	kubeApiPodName := fmt.Sprintf("kube-apiserver-%s", node.Name)
	for _, p := range pods.Items {
		if p.Name == kubeApiPodName {
			endpoint, ok := p.Annotations[kubeadmconst.KubeAPIServerAdvertiseAddressEndpointAnnotationKey]
			if !ok {
				return "", 0, fmt.Errorf("could not find endpoint annotation %s on node %s", kubeadmconst.KubeAPIServerAdvertiseAddressEndpointAnnotationKey, node.Name)
			}

			return parseEndpoint(endpoint)
		}
	}

	return "", 0, fmt.Errorf("could not find kube-apiserver pod for node %s", node.Name)
}

// virtualIp handles the migration from the self-managed keepalived and nginx
// configuration to one that executes in the cluster.
func virtualIp(restConfig *rest.Config, client kubernetes.Interface, kubeConfigPath string, nodes *v1.NodeList) error {
	//  If the daemonset based VIP config updater is already deployed,
	// then this must already be done.
	haRelease, err := getRelease(constants.HAMonitorRelease, constants.HAMonitorNamespace, kubeConfigPath)
	if err != nil {
		return err
	} else if haRelease != nil {
		log.Debugf("Virtual IP monitor is already installed")
		return nil
	}

	cc, err := k8s.GetKubeadmClusterConfiguration(client)
	if err != nil {
		return err
	}

	apiHost, apiHostPort, err := parseEndpoint(cc.ControlPlaneEndpoint)
	if err != nil {
		return err
	}

	// If the control plane endpoint address is assigned to one of the
	// nodes, then it cannot be a VIP.  This is normal for a cluster
	// with a single control plane node, but in some cases for simple
	// testing it can happen with a multi control plane node cluster
	// as well.
	for _, n := range nodes.Items {
		if !k8s.IsControlPlaneNode(&n) {
			continue
		}

		for _, a := range n.Status.Addresses {
			if apiHost == a.Address {
				log.Debugf("Node %s advertises the control plane endpoint %s", n.Name, apiHost)
				return nil
			}
		}

		// If the provider is one that can have a VIP configuration,
		// then it is possible for the nodes to have one.
		if isVipableProvider(n.Spec.ProviderID) {
			log.Debugf("Node %s has provider id that can host a virtual IP: %s", n.Name, n.Spec.ProviderID)
			continue
		}

		// If the provider is one that would not typically have a VIP
		// configuration, but the cluster is not from CAPI, then it
		// might have one anyway.
		_, ok := n.Annotations[constants.CAPIClusterNameAnnotation]
		if !ok {
			continue
		}

		// There is no reason to believe that this cluster uses
		// a VIP configuration.  Bail now to avoid the expensive
		// checks that are to follow.
		return nil
	}

	// Check to see if a VIP is configured by seeing if the control
	// plane endpoint is present in the keepalived.conf
	for _, n := range nodes.Items {
		if !k8s.IsControlPlaneNode(&n) {
			continue
		}

		kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, constants.OCNESystemNamespace, nil, false)
		if err != nil {
			return err
		}

		kcConfig.Streams.Out = bytes.NewBuffer([]byte{})

		err = script.RunScript(client, kcConfig, n.Name, constants.OCNESystemNamespace, "check-ips", GetKeepalivedConf, []v1.EnvVar{})
		if err != nil {
			return err
		}

		err = k8s.DeletePod(client, constants.OCNESystemNamespace, fmt.Sprintf("check-ips-%s", n.Name))
		if err != nil {
			return err
		}

		keepalivedConf := kcConfig.Streams.Out.(*bytes.Buffer).String()
		log.Debugf("Looking for %s", apiHost)
		log.Debugf("  in:")
		log.Debugf(keepalivedConf)
		if strings.Contains(keepalivedConf, apiHost) {
			log.Debugf("Node %s manages a virtual IP", n.Name)
			continue
		}

		// If keepalived and nginx are not enabled, then the node
		// must node have a VIP and by extension there is nothing
		// to upgrade.
		log.Debugf("Node %s does not use a virtual IP", n.Name)
		return nil
	}

	// At this point it is known that an upgrade is required.  Now comes
	// the hard part.  Upgrading requires the following:
	//  - update keepalived check script
	//  - stop keepalived refresh service
	//  - stop nginx refresh service
	//  - delete kubeconfigs
	//  - deploy path and refresh trigger units for keepalived/nginx
	//  - install monitor daemonset to cluster
	kubeSystemPods, err := k8s.GetPods(client, constants.KubeNamespace)
	if err != nil {
		return err
	}

	for _, n := range nodes.Items {
		if !k8s.IsControlPlaneNode(&n) {
			continue
		}

		_, endpointPort, err := getEndpointForNode(&n, kubeSystemPods)
		if err != nil {
			return err
		}

		updateScript, err := generateVipUpdateScript(apiHostPort, endpointPort, apiHost)
		if err != nil {
			return err
		}
		log.Debugf("update script for node %s is:", n.Name)
		log.Debugf(updateScript)

		kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, constants.OCNESystemNamespace, nil, false)
		if err != nil {
			return err
		}

		err = script.RunScript(client, kcConfig, n.Name, constants.OCNESystemNamespace, "upgrade-vips", updateScript, []v1.EnvVar{})
		if err != nil {
			return err
		}

		_ = k8s.DeletePod(client, constants.OCNESystemNamespace, fmt.Sprintf("check-update-%s", n.Name))
	}

	err = install.InstallApplications([]install.ApplicationDescription{
		{
			Application: &types.Application{
				Name: constants.HAMonitorChart,
				Namespace: constants.HAMonitorNamespace,
				Release: constants.HAMonitorRelease,
				Version: constants.HAMonitorVersion,
				Catalog: catalog.InternalCatalog,
				Config: map[string]interface{}{
					"apiAddress": apiHost,
					"apiPort": strconv.FormatUint(uint64(apiHostPort), 10),
				},
			},
		},
	}, kubeConfigPath, false)

	// If this is an OLVM cluster, crack the ignition contents and
	// replace the contents so it matches the result of the process
	// above.

	return nil
}

// updateFuncs is an ordered list of update functions to run.
var updateFuncs = []func(*rest.Config, kubernetes.Interface, string, *v1.NodeList)error{
	oneThirtyAndLower,
	virtualIp,
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

// Custom time type for non-standard format
const ctLayout = "2006-01-02 15:04:05 -0700"
type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	// Remove quotes
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	t, err := time.Parse(ctLayout, s)
	if err != nil {
		return err
	}
	ct.Time = t
	return nil
}

// Struct for your JSON
type BootUpdateTimestamps struct {
	BootTimestamp   CustomTime `json:"boot_timestamp"`
	UpdateTimestamp CustomTime `json:"update_timestamp"`
}

// IsUpdateAvailable returns true if an update is available on that node
func IsUpdateAvailable(node *v1.Node, kubeClient kubernetes.Interface, restConfig *rest.Config, kubeConfigPath string) (bool, error) {
	if node.Annotations != nil {
		v, ok := node.Annotations[constants.OCNEAnnoUpdateAvailable]
		if ok && strings.ToLower(v) == "true" {
			return true, nil
		}
	}

	// Now into the obscure cases.
	//
	// Kubernetes 1.32 introduced a change to the permissions assigned
	// to the kubelet service account.  Notably, it can no longer list
	// nodes.  Versions of OCK prior to recent 1.32 builds use a selector
	// to annotate the node indicating that an update is available.  That
	// command fails because selectors require listing nodes.  This issue
	// is limited to OCK instances running Kubernetes 1.32.5/7.  Do a more
	// intensive check for that case.
	if node.Status.NodeInfo.KubeletVersion == "v1.32.7+1.el8" || node.Status.NodeInfo.KubeletVersion == "v1.32.5+1.el8"  {
		kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, constants.OCNESystemNamespace, nil, false)
		if err != nil {
			return false, err
		}

		kcConfig.Streams.Out = bytes.NewBuffer([]byte{})
		err = script.RunScript(kubeClient, kcConfig, node.Name, constants.OCNESystemNamespace, "check-update-127", CheckNodeUpdate, []v1.EnvVar{})
		if err != nil {
			return false, err
		}
		err = k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, fmt.Sprintf("check-update-%s", node.Name))
		if err != nil {
			return false, err
		}
		ockInfo := kcConfig.Streams.Out.(*bytes.Buffer)

		timestamps := BootUpdateTimestamps{}
		err = json.Unmarshal(ockInfo.Bytes(), &timestamps)
		if err != nil {
			return false, err
		}

		return timestamps.BootTimestamp.Time.Before(timestamps.UpdateTimestamp.Time), nil
	}

	return false, nil
}

func IsUpdateAvailableByName(name string, kubeClient kubernetes.Interface, restConfig *rest.Config, kubeConfigPath string) (bool, error) {
	node, err := k8s.GetNode(kubeClient, name)
	if err != nil {
		return false, err
	}
	return IsUpdateAvailable(node, kubeClient, restConfig, kubeConfigPath)
}

