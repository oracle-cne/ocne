// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cluster"
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/add"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/unix"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/release"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/api/core/v1"
)

const (
	uiServicePort = "8443"
	uiTargetPort  = "443"

	createTokenFormatText       = "Run the following command to create an authentication token to access the UI:\n    %s"
	createTokenStanzaFormatText = "%skubectl create token ui -n %s"
)

// getTagForApplication returns the "best" tag for an application.  In general,
// the best tag is "current".  Older installs might not have this tag, so
// revert to whatever is available.
func getTagForApplication(img string, bestTag string, legacyTag string, node *v1.Node) (string, error) {
	tag := ""
	log.Debugf("Checking %s on %s", img, node.Name)
	bestImg, haveBest, haveLegacy := k8s.GetImageCandidate(img, bestTag,  legacyTag, node)
	if haveBest {
		tag = bestTag
	} else if haveLegacy {
		tag = legacyTag
	} else {
		imgInfo, err := image.SplitImage(bestImg)
		if err != nil {
			// In practice this is not possible,
			// since the image was sourced from
			// Kubernetes and has already been
			// vetted.  Still, stranger things
			// have happened.
			return "", err
		}
		tag = imgInfo.Tag
		log.Warn("Control plane node %s has an unexpected image.  Using %s", node.Name, tag)
	}
	return tag, nil
}

// getImageTag gets an image from a node.  It will always return a tagged image.
func getImageTag(img string, node *v1.Node) (string, error) {
	for _, cis := range node.Status.Images {
		for _, name := range cis.Names {
			log.Debugf("Checking %s against %s on %s", name, img, node.Name)
			if !strings.HasPrefix(name, img) {
				continue
			}

			imgInfo, err := image.SplitImage(name)
			if err != nil {
				return "", err
			}

			if imgInfo.Tag != "" {
				return imgInfo.Tag, nil
			}
		}
	}
	return "", fmt.Errorf("did not have the %s image on node %s", img, node.Name)
}

// Start starts a cluster based on the given configuration and returns the
// canonical kubeconfig
func Start(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
	// Check to see if the cluster already exists.  If it does, make
	// sure it is the "same cluster" for some appropriate definition
	// of same cluster.
	// The none provider doesn't utilize the cache, so these cache checks are skipped for the none provider

	clusterCache, err := cache.GetCache()
	if err != nil {
		return "", err
	}
	if clusterConfig.Provider != constants.ProviderTypeNone {
		cachedClusterConfig := clusterCache.Get(clusterConfig.Name)
		if cachedClusterConfig != nil {
			if cachedClusterConfig.ClusterConfig.Provider != clusterConfig.Provider {
				return "", fmt.Errorf("the provider of the existing cluster is %s. The target provider is %s", cachedClusterConfig.ClusterConfig.Provider, clusterConfig.Provider)
			}
			if cachedClusterConfig != nil && cachedClusterConfig.ClusterConfig.KubeVersion != clusterConfig.KubeVersion {
				return "", fmt.Errorf("the Kubernetes version of the existing cluster is %s. The target Kubernetes version is %s", cachedClusterConfig.ClusterConfig.KubeVersion, clusterConfig.KubeVersion)
			}
		}
	}
	infoFuncWait := logutils.Info
	infoFunc := log.Info
	infofFunc := log.Infof
	if config.Quiet {
		infoFuncWait = func(string) {}
		infoFunc = func(args ...interface{}) {}
		infofFunc = func(s string, a ...any) {}
	}
	drv, err := driver.CreateDriver(config, clusterConfig)
	if err != nil {
		return "", err
	}
	defer drv.Close()

	wasRunning, skipInstall, err := drv.Start()
	if err != nil {
		return "", err
	}

	localKubeConfig := drv.GetKubeconfigPath()

	if !wasRunning {
		clusterCache, err = cache.GetCache()
		if err != nil {
			return "", err
		}
		err = clusterCache.Add(clusterConfig, localKubeConfig)
		if err != nil {
			return localKubeConfig, err
		}
	}

	if skipInstall {
		return localKubeConfig, err
	}

	_, kubeClient, err := client.GetKubeClient(localKubeConfig)
	if err != nil {
		return localKubeConfig, err
	}

	// There may be cases where an old version of OCK
	// is being used.  In that case, Flannel might not
	// have the floating "current" tag.  If so, just
	// use whatever is available, preferring what
	// was installed in 2.0.x.  Pick something from
	// a control plane node since there will always
	// be one.
	cpNodes, err := k8s.WaitForControlPlaneNodes(kubeClient)
	if err != nil {
		return localKubeConfig, err
	}

	// There has to be one control plane node to answer the query
	// so this list will always have at least one element.
	cpNode := &cpNodes.Items[0]

	// Install charts that are baked in to this application and from
	// the Oracle catalog.
	//
	// kube-proxy is forcibly installed to account for old cluster
	// descriptions that use kubeadm to deploy kube-proxy.  Same
	// with coredns.  Old versions of OCK may not have the new
	// standard "current" tags.  It is necessary to install with
	// old configurations and then update them later.  For CoreDNS,
	// the version that was installed is known.  For kube-proxy, it
	// can vary by Kubernetes version.  Derive the legacy kube-proxy
	// image tag from another core Kubernets container image.
	coreDnsTag, err := getTagForApplication(constants.CoreDNSImage, constants.CoreDNSTag, constants.CoreDNSLegacyTag, cpNode)
	if err != nil {
		return localKubeConfig, err
	}

	kubeProxyLegacyTag, err := getImageTag(constants.KubeAPIServerImage, cpNode)
	if err != nil {
		return localKubeConfig, err
	}
	kubeProxyTag, err := getTagForApplication(constants.KubeProxyImage, constants.KubeProxyTag, kubeProxyLegacyTag, cpNode)
	if err != nil {
		return localKubeConfig, err
	}
	applications := []install.ApplicationDescription{
		install.ApplicationDescription{
			Force: true,
			Application: &types.Application{
				Name:      constants.CoreDNSChart,
				Namespace: constants.CoreDNSNamespace,
				Release:   constants.CoreDNSRelease,
				Version:   constants.CoreDNSVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"image": map[string]interface{}{
						"tag": coreDnsTag,
					},
				},
			},
		},
		install.ApplicationDescription{
			Force: true,
			Application: &types.Application{
				Name:      constants.KubeProxyChart,
				Namespace: constants.KubeProxyNamespace,
				Release:   constants.KubeProxyRelease,
				Version:   constants.KubeProxyVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"image": map[string]interface{}{
						"tag": kubeProxyTag,
					},
					"apiServer": map[string]interface{}{
						"host": drv.GetKubeAPIServerAddress(),
						"port": clusterConfig.KubeAPIServerBindPort,
					},
					"config": map[string]interface{}{
						"mode": clusterConfig.KubeProxyMode,
						"clusterCIDR": clusterConfig.ServiceSubnet,
					},
				},
			},
		},
	}

	if clusterConfig.Provider != constants.ProviderTypeNone {
		switch clusterConfig.CNI {
		case "", constants.CNIFlannel:
			log.Debugf("Flannel will be installed as the CNI")


			// If there are no control plane nodes, then it the
			// query for them will fail because nobody can answer.
			// In practice, it's not possible to have a zero length
			// node list.
			tag, err := getTagForApplication(constants.CNIFlannelImage, constants.CNIFlannelImageTag, constants.CNIFlannelLegacyTag, cpNode)
			if err != nil {
				return localKubeConfig, err
			}

			args := []string{
				"--ip-masq",
				"--kube-subnet-mgr",
			}
			ifaces := drv.DefaultCNIInterfaces()
			for _, i := range ifaces {
				args = append(args, fmt.Sprintf("--iface=%s", i))
			}
			applications = append(applications, install.ApplicationDescription{
				Application: &types.Application{
					Name:      constants.CNIFlannelChart,
					Namespace: constants.CNIFlannelNamespace,
					Release:   constants.CNIFlannelRelease,
					Version:   constants.CNIFlannelVersion,
					Catalog:   catalog.InternalCatalog,
					Config: map[string]interface{}{
						"podCidr": clusterConfig.PodSubnet,
						"flannel": map[string]interface{}{
							"args": args,
							"image": map[string]interface{}{
								"tag": tag,
							},
						},
					},
				},
			})
		case constants.CNINone:
			// If no CNI is installed, it is not possible for the UI
			// to start.  Don't wait for it.
			config.AutoStartUI = "false"
			log.Debugf("No CNI will be installed")
		}
	}

	if !clusterConfig.Headless {
		log.Debugf("Installing UI")

		// If there are no control plane nodes, then it the
		// query for them will fail because nobody can answer.
		// In practice, it's not possible to have a zero length
		// node list.
		tag, err := getTagForApplication(constants.UIImage, constants.UIImageTag, constants.UILegacyTag, cpNode)
		if err != nil {
			return localKubeConfig, err
		}

		// Determine if the image registry needs to be overridden
		helmOverride := map[string]interface{}{}
		if clusterConfig.Provider == constants.ProviderTypeNone &&
			clusterConfig.Registry != constants.ContainerRegistry {
			helmOverride = map[string]interface{}{
				"image": map[string]interface{}{
					"registry": clusterConfig.Registry,
				},
				"initContainers": []map[string]interface{}{
					{
						"name":            constants.UIInitContainer,
						"image":           fmt.Sprintf("%s/olcne/ui-plugins:%s", clusterConfig.Registry, constants.UIPluginsVersion),
						"imagePullPolicy": "IfNotPresent",
						"command":         []string{"/bin/sh", "-c", "mkdir -p /build/plugins && cp -r /headlamp-plugins/* /build/plugins/"},
						"volumeMounts": []map[string]interface{}{
							{
								"name":      "ui-plugins",
								"mountPath": "/build/plugins",
							},
						},
					},
				},
			}
		}

		// Add the tag.  There might be overrides set by the
		// previous block, so make sure to precision insert
		// the tag.
		var imgOvr map[string]interface{}
		imgOvrIface, ok := helmOverride["image"]
		if !ok {
			imgOvr = map[string]interface{}{}
			helmOverride["image"] = imgOvr
		} else {
			imgOvr, ok = imgOvrIface.(map[string]interface{})
			if !ok {
				return localKubeConfig, fmt.Errorf("internal error: Helm overrides for the UI are corrupt")
			}
		}
		imgOvr["tag"] = tag

		applications = append(applications, install.ApplicationDescription{
			PreInstall: func() error {
				err := cluster.CreateCert(kubeClient, constants.UINamespace)
				return err
			},
			Application: &types.Application{
				Name:      constants.UIChart,
				Namespace: constants.UINamespace,
				Release:   constants.UIRelease,
				Version:   constants.UIVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    helmOverride,
			},
		})
	} else {
		config.AutoStartUI = "false"
	}

	if clusterConfig.Catalog {
		log.Debugf("Installing Oracle Catalog")

		// Determine if the image registry needs to be overridden
		helmOverride := map[string]interface{}{}
		if clusterConfig.Provider == constants.ProviderTypeNone &&
			clusterConfig.Registry != constants.ContainerRegistry {
			helmOverride = map[string]interface{}{
				"image": map[string]interface{}{
					"registry": clusterConfig.Registry,
				},
			}
		}
		applications = append(applications, install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CatalogChart,
				Namespace: constants.CatalogNamespace,
				Release:   constants.CatalogRelease,
				Version:   constants.CatalogVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    helmOverride,
			},
		})
	}

	// Get all the installed applications
	releases, err := getAllReleases(localKubeConfig)
	if err != nil {
		return localKubeConfig, err
	}

	// Get the list of applications to install
	appsToInstall := getAppsToInstall(applications, releases)
	err = install.InstallApplications(appsToInstall, localKubeConfig, config.Quiet)
	if err != nil {
		return localKubeConfig, err
	}

	// Get all the external catalogs to add
	catalogsToAdd, err := getCatalogsToAdd(localKubeConfig, clusterConfig)
	if err != nil {
		return localKubeConfig, err
	}

	for _, c := range catalogsToAdd {
		// Generate a service name from the catalog name
		svcName := strings.ReplaceAll(c.Name, " ", "")
		svcName = strings.ToLower(svcName)
		if len(svcName) >= 64 {
			svcName = svcName[:63]
		}
		err = add.Add(localKubeConfig, svcName, c.Namespace, c.URI, c.Protocol, c.Name)
		if err != nil {
			return localKubeConfig, err
		}
	}

	// Install any other configured applications
	applications = []install.ApplicationDescription{}
	for i := range clusterConfig.Applications {
		log.Debugf("Queueing application %s with release name %s queued up", clusterConfig.Applications[i].Name, clusterConfig.Applications[i].Release)
		applications = append(applications, install.ApplicationDescription{
			Application: &clusterConfig.Applications[i],
		})
	}

	// Get the list of applications to install from the cluster configuration and install them. If any of
	// the applications come from the default app catalog, wait for the app catalog to be ready.
	appsToInstall = getAppsToInstall(applications, releases)
	for _, app := range appsToInstall {
		if app.Application == nil {
			continue
		}
		if len(app.Application.Catalog) == 0 || app.Application.Catalog == constants.DefaultCatalogName {
			if err := catalog.WaitForInternalCatalogInstall(kubeClient, infoFuncWait); err != nil {
				return localKubeConfig, err
			}
			break
		}
	}
	err = install.InstallApplications(appsToInstall, localKubeConfig, config.Quiet)
	if err != nil {
		return localKubeConfig, err
	}

	// Do any cluster stuff that requires other applications to be installed.  For example
	// some clusters require things like CNIs to be installed for the necessary bits
	// to work.
	err = drv.PostStart()
	if err != nil {
		return localKubeConfig, err
	}
	drv.Close()

	// success - print out directions on what to do next
	infoFunc("Kubernetes cluster was created successfully")

	// Determine if port forwards need to be setup and a browser started
	if config.AutoStartUI == "true" {
		if err := autoStartUI(kubeClient, localKubeConfig, infoFuncWait); err != nil {
			return localKubeConfig, err
		}
	}

	if !clusterConfig.Headless {
		accessUI := fmt.Sprintf("To access the UI, first do kubectl port-forward to allow the browser to access the UI.\nRun the following command, then access the UI from the browser using via https://localhost:%s", uiServicePort)
		portForward1 := fmt.Sprintf("    kubectl port-forward -n %s service/%s %s:%s", constants.OCNESystemNamespace, constants.UIServiceName, uiServicePort, uiTargetPort)
		createToken := fmt.Sprintf(createTokenFormatText, fmt.Sprintf(createTokenStanzaFormatText, "", constants.OCNESystemNamespace))
		postInstallMsg := fmt.Sprintf("Post install information:\n\n%s\n%s\n%s\n%s", drv.PostInstallHelpStanza(), accessUI, portForward1, createToken)

		infofFunc("%s", postInstallMsg)
	} else {
		infofFunc("Post install information:\n\n%s\n", drv.PostInstallHelpStanza())
	}
	return localKubeConfig, nil
}

func autoStartUI(client kubernetes.Interface, localKubeConfig string, infoFunc func(string)) error {
	var err error
	haveError := logutils.WaitFor(infoFunc, []*logutils.Waiter{
		{
			Message: "Waiting for the UI to be ready",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(client, "ocne-system", "ui", 1)
			},
		},
	})
	if haveError {
		return fmt.Errorf("timed out starting UI")
	}

	// Setup port-forward to expose UI outside the cluster
	port, err := k8s.PortForwardToService(localKubeConfig, k8stypes.NamespacedName{Name: constants.UIServiceName, Namespace: constants.OCNESystemNamespace}, uiTargetPort)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://localhost:%d", port)

	// Start a browser window
	switch runtime.GOOS {
	case "linux":
		err = unix.NewCmdExecutor("xdg-open", url).Start()
	case "darwin":
		err = unix.NewCmdExecutor("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return err
	}

	// Prompt the user to confirm when to exit
	var userInput string
	for {
		createTokenStanza := fmt.Sprintf(createTokenStanzaFormatText, fmt.Sprintf("KUBECONFIG='%s' ", localKubeConfig), constants.OCNESystemNamespace)
		fmt.Printf("\n%s\nBrowser window opened, enter 'y' when ready to exit: ", fmt.Sprintf(createTokenFormatText, createTokenStanza))
		fmt.Scanln(&userInput)
		if strings.ToLower(userInput) == "y" {
			break
		}
	}
	fmt.Println()
	return nil
}

// getCatalogsToAdd returns the list of catalogs which are not installed
func getCatalogsToAdd(kubeConfig string, clusterConfig *types.ClusterConfig) ([]types.Catalog, error) {
	var catalogToInstall []types.Catalog
	allCatalogs, err := ls.Ls(kubeConfig)
	if err != nil {
		return nil, err
	}

	for index := range clusterConfig.Catalogs {
		catalogInstalled := false
		for _, catalog := range allCatalogs {
			if catalog.CatalogName == clusterConfig.Catalogs[index].Name &&
				catalog.ServiceNsn.Namespace == clusterConfig.Catalogs[index].Namespace {
				catalogInstalled = true
				break
			}
		}
		if !catalogInstalled {
			log.Debugf("The catalog with the name %s is not yet deployed", clusterConfig.Catalogs[index].Name)
			catalogToInstall = append(catalogToInstall, clusterConfig.Catalogs[index])
		}
	}

	if clusterConfig.CommunityCatalog {
		catalogToInstall = append(clusterConfig.Catalogs, catalog.NewCommunityCatalog())
	}

	return catalogToInstall, nil
}

// getAllReleases returns the release information from all the namespaces
func getAllReleases(kubeConfig string) ([]*release.Release, error) {
	kubeInfo, err := client.CreateKubeInfo(kubeConfig)
	if err != nil {
		return nil, err
	}
	return helm.GetReleasesAllNamespaces(kubeInfo)
}

// getAppsToInstall returns the list of applications, which are not installed in the cluster
func getAppsToInstall(allApps []install.ApplicationDescription, releases []*release.Release) []install.ApplicationDescription {
	var appsToInstall []install.ApplicationDescription
	for index := range allApps {
		appInstalled := false
		for _, rel := range releases {
			if rel.Name == allApps[index].Application.Release && rel.Namespace == allApps[index].Application.Namespace {
				log.Debugf("The application with release name %s is already deployed", rel.Name)
				appInstalled = true
				break
			}
		}
		if !appInstalled {
			appsToInstall = append(appsToInstall, allApps[index])
		}
	}
	return appsToInstall
}
