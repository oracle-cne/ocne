// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package join

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	clustertypes "github.com/oracle-cne/ocne/pkg/cluster/types"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/script"
)

const copyPodPrefix = "copy-files"

type JoinOptions struct {
	Config             *types.Config
	ClusterConfig      *types.ClusterConfig
	KubeConfigPath     string
	ControlPlaneNodes  int
	WorkerNodes        int
	Node               string
	DestKubeConfigPath string
	RoleControlPlane   bool
}

func Join(options *JoinOptions) error {
	if options.Node != "" && options.DestKubeConfigPath != "" {
		return joinNodeToCluster(options)
	}

	drv, err := driver.CreateDriver(options.Config, options.ClusterConfig)
	if err != nil {
		return err
	}

	err = drv.Join(options.KubeConfigPath, options.ControlPlaneNodes, options.WorkerNodes)
	return err
}

// joinNodeToCluster moves a node from one cluster to a destination cluster identified by destKubeconfigPath.
func joinNodeToCluster(options *JoinOptions) error {
	restConfig, kubeClient, err := client.GetKubeClient(options.KubeConfigPath)
	if err != nil {
		return err
	}

	// make sure the node we are trying to migrate exists in the source cluster
	if _, err := k8s.GetNode(kubeClient, options.Node); err != nil {
		return err
	}

	_, destKubeClient, err := client.GetKubeClient(options.DestKubeConfigPath)
	if err != nil {
		return err
	}

	// get the control plane endpoint from the destination cluster
	cpEndpoint, err := getControlPlaneEndpoint(destKubeClient)
	if err != nil {
		return err
	}

	// create a join token and push it to the destination cluster
	token, certHashes, err := k8s.CreateJoin(options.DestKubeConfigPath)
	if err != nil {
		return err
	}

	// generate a kubeadm.conf file that will join the node to the destination cluster
	parts := strings.Split(cpEndpoint, ":")
	if len(parts) != 2 {
		return fmt.Errorf("Malformed control plane endpoint: %s", cpEndpoint)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}
	cj := &ignition.ClusterJoin{
		KubeAPIServerIP:   parts[0],
		KubeAPIBindPort:   uint16(port),
		JoinToken:         token,
		KubePKICertHashes: certHashes,
	}

	var enableServices []string

	if options.RoleControlPlane {
		// if the user wants the node to be a control plane in the destination cluster, there's more config to do
		if err := configureControlPlane(destKubeClient, options.DestKubeConfigPath, cj, options.ClusterConfig.Registry); err != nil {
			return err
		}

		// may also need to configure keepalive for HA
		if enableServices, err = configureHA(options, cj); err != nil {
			return err
		}
	}

	joinYaml, err := ignition.GenerateKubeadmJoinYaml(cj)
	if err != nil {
		return err
	}

	// run a script in a pod on the node to update the config, perform a "kubeadm reset", and join
	// the node to the destination cluster
	if err := updateNode(restConfig, kubeClient, options.KubeConfigPath, options.Node, joinYaml, enableServices); err != nil {
		return err
	}

	// the last thing we do is delete the node from the source cluster (otherwise it still shows up when fetching nodes)
	// NOTE: this can fail for several reasons (for example, if this is the last/only control plane node), so
	// make this a best effort and ignore errors
	kubeClient.CoreV1().Nodes().Delete(context.TODO(), options.Node, metav1.DeleteOptions{})
	return nil
}

// getControlPlaneEndpoint fetches the control plane endpoint from the kubeadm-config ConfigMap.
func getControlPlaneEndpoint(kubeClient kubernetes.Interface) (string, error) {
	cm, err := kubeClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(context.TODO(), kubeadmconst.KubeadmConfigConfigMap, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	ccYaml := cm.Data["ClusterConfiguration"]
	cc := &ignition.ClusterConfig{}
	if err = yaml.Unmarshal([]byte(ccYaml), cc); err != nil {
		return "", err
	}

	return cc.ControlPlaneEndpoint, nil
}

// updateNode creates a temporary pod and runs a script to reconfigure the node so that it joins the destination cluster.
func updateNode(restConfig *rest.Config, kubeClient kubernetes.Interface, kubeconfigPath string, node string, joinYaml string, enableServices []string) error {
	namespace := constants.OCNESystemNamespace

	// ensure the Namespace exists
	if err := k8s.CreateNamespaceIfNotExists(kubeClient, namespace); err != nil {
		return err
	}

	// get config needed to use kubectl
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeconfigPath, namespace, []string{}, false)
	if err != nil {
		return err
	}

	log.Info("Updating node with join configuration")
	if err := script.RunScript(kubeClient, kcConfig, node, namespace, "join-node", updateNodeScript, []corev1.EnvVar{
		{Name: "JOIN_CONFIG", Value: joinYaml},
		{Name: "ENABLE_SERVICES", Value: strings.Join(enableServices, " ")},
	}); err != nil {
		return err
	}

	log.Infof("Node %s successfully updated", node)

	return nil
}

// configureControlPlane pushes upload certificates and a new upload key to the destination cluster and adds
// the right bits to the ClusterJoin struct so that the node will join the destination cluster as a
// control plane
func configureControlPlane(kubeClient kubernetes.Interface, kubeconfig string, cj *ignition.ClusterJoin, registry string) error {
	// get the API server pods from the destination cluster so we can determine the API server listen port
	pods, err := k8s.GetPodsBySelector(kubeClient, metav1.NamespaceSystem, "component="+kubeadmconst.KubeAPIServer)
	if err != nil {
		return err
	}

	if pods == nil || len(pods.Items) == 0 {
		return fmt.Errorf("Unable to fetch API server pods from destination cluster")
	}

	var listenPort int

	// just use the first pod in the list, and get the secure-port from the container args
	for _, container := range pods.Items[0].Spec.Containers {
		if container.Name == kubeadmconst.KubeAPIServer {
			for _, cmd := range container.Command {
				if strings.HasPrefix(cmd, "--secure-port=") {
					parts := strings.Split(cmd, "=")
					if listenPort, err = strconv.Atoi(parts[1]); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	if listenPort == 0 {
		return fmt.Errorf("Unable to find API server listen port")
	}

	// generate a new key and upload certs and the key to the destination cluster
	uploadCertificateKey, err := util.CreateUploadCertificateKey()
	if err != nil {
		return err
	}

	log.Info("Uploading control plane certificates to destination cluster")
	err = k8s.UploadCertificates(kubeconfig, uploadCertificateKey, registry)
	if err != nil {
		return err
	}

	// set control plane fields in the ClusterJoin struct so the extra control plane config appears in the JoinConfiguration
	cj.Role = clustertypes.ControlPlaneRole
	cj.UploadCertificateKey = uploadCertificateKey
	// setting this to true causes GenerateKubeadmJoinYaml to use the correct API bind port field for the control plane config
	cj.InternalLB = true
	cj.KubeAPIBindPortAlt = uint16(listenPort)

	return nil
}

// configureHA configures the migrating node for HA if the destination cluster is using a virtual IP. If the
// destination cluster is not using a virtual IP, then this is a no-op. This function copies files from a
// control plane node on the destination cluster and files from the migrating node, and uses collected data
// to generate all of the keepalive and nginx configuration. Those files are then copied to the migrating node.
func configureHA(options *JoinOptions, cj *ignition.ClusterJoin) ([]string, error) {
	tmpDir, err := file.CreateOcneTempDir("files")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	vip := options.ClusterConfig.VirtualIp
	bindPort := options.ClusterConfig.KubeAPIServerBindPort
	bindPortAlt := options.ClusterConfig.KubeAPIServerBindPortAlt

	// if the user has specified the virtual IP, then we'll just use that, otherwise we have to get files from
	// a control plane node on the destination cluster and parse the data
	if vip == "" {
		keepaliveTemplateFile, err := getFilesFromDestCluster(options, tmpDir)
		if err != nil {
			return nil, err
		}

		if vip == "" {
			// read the template file and parse the virtual IP
			vip, err = parseVirtualIP(keepaliveTemplateFile)
			if err != nil {
				return nil, err
			}
			if vip == "" {
				// no virtual IP configured so this is not a keepalive situation, nothing else to do
				return nil, nil
			}
		}
	}

	// if user has not specified ports, use the ports we derived earlier
	if bindPort == 0 {
		bindPort = cj.KubeAPIBindPort
	}
	if bindPortAlt == 0 {
		bindPortAlt = cj.KubeAPIBindPortAlt
	}

	// get the network interface and optional proxy config from the node we are migrating
	bootstrapConfFile, proxyConfFile, err := getFilesFromMigratingNode(options, tmpDir)
	if err != nil {
		return nil, err
	}

	netInterface, err := parseNetworkInterface(bootstrapConfFile)
	if err != nil {
		return nil, err
	}
	proxy, err := parseProxyConfig(proxyConfFile)
	if err != nil {
		return nil, err
	}

	// we have collected all the data, now generate the keepalive files and copy them to the node we are migrating
	return copyHAFilesToNode(options, bindPort, bindPortAlt, vip, proxy, netInterface)
}

// getFilesFromDestCluster copies files from a control plane node on the destination cluster.
func getFilesFromDestCluster(options *JoinOptions, tmpDir string) (string, error) {
	restConfig, kubeClient, err := client.GetKubeClient(options.DestKubeConfigPath)
	if err != nil {
		return "", err
	}

	// copy the keepalived config template from one of the destination cluster control plane nodes - if this file
	// does not exist then we're not using a virtual IP
	nodes, err := k8s.GetControlPlaneNodes(kubeClient)
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("Could not get control plane nodes from destination cluster")
	}

	cc, err := startAdminPod(restConfig, kubeClient, options.DestKubeConfigPath, nodes.Items[0].Name, options.ClusterConfig.Registry)
	defer k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, copyPodPrefix+"-"+nodes.Items[0].Name)
	if err != nil {
		return "", err
	}

	localTemplatePath := filepath.Join(tmpDir, filepath.Base(ignition.KeepAlivedConfigTemplatePath))
	cc.FilePaths = append(cc.FilePaths, kubectl.FilePath{RemotePath: "/hostroot" + ignition.KeepAlivedConfigTemplatePath, LocalPath: localTemplatePath})

	if err = kubectl.CopyFilesFromPod(cc, "Copying files from destination cluster"); err != nil {
		return "", err
	}

	return localTemplatePath, nil
}

// getFilesFromMigratingNode copies files from the node we are migrating.
func getFilesFromMigratingNode(options *JoinOptions, tmpDir string) (string, string, error) {
	restConfig, kubeClient, err := client.GetKubeClient(options.KubeConfigPath)
	if err != nil {
		return "", "", err
	}

	cc, err := startAdminPod(restConfig, kubeClient, options.KubeConfigPath, options.Node, options.ClusterConfig.Registry)
	defer k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, copyPodPrefix+"-"+options.Node)
	if err != nil {
		return "", "", err
	}

	const bootstrapConfPath = "/etc/systemd/system/ocne.service.d/bootstrap.conf"
	const proxyConfPath = "/etc/systemd/system/ocne-update.service.d/proxy.conf"

	localBootstrapConfPath := filepath.Join(tmpDir, filepath.Base(bootstrapConfPath))
	cc.FilePaths = append(cc.FilePaths, kubectl.FilePath{RemotePath: "/hostroot" + bootstrapConfPath, LocalPath: localBootstrapConfPath})
	localProxyConfPath := filepath.Join(tmpDir, filepath.Base(proxyConfPath))
	cc.FilePaths = append(cc.FilePaths, kubectl.FilePath{RemotePath: "/hostroot" + proxyConfPath, LocalPath: localProxyConfPath})

	if err = kubectl.CopyFilesFromPod(cc, "Copying files from migrating node"); err != nil {
		return "", "", err
	}

	return localBootstrapConfPath, localProxyConfPath, nil
}

// parseVirtualIP parses the virtual IP from the keepalived template.
func parseVirtualIP(file string) (string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		// if the file does not exist, then this is not a virtual IP scenario
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	regex := regexp.MustCompile(`virtual_ipaddress {\s*(.*)\s*}`)
	match := regex.FindStringSubmatch(string(b))
	if len(match) < 2 {
		return "", fmt.Errorf("Unable to parse virtual IP from keepalive configuration template")
	}

	return match[1], nil
}

// parseNetworkInterface parses the network interface from the bootstrap configuration file.
func parseNetworkInterface(file string) (string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`NET_INTERFACE=(.*)\s*$`)
	match := regex.FindStringSubmatch(string(b))
	if len(match) < 2 {
		return "", fmt.Errorf("Unable to parse network interface from bootstrap configuration file")
	}

	return match[1], nil
}

// parseProxyConfig parses the optional proxy configuration (taken from optional proxy configuration
// on the ocne-update service).
func parseProxyConfig(file string) (*types.Proxy, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		// if the file does not exist, then the user has not configured a proxy
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	proxy := &types.Proxy{}

	regex := regexp.MustCompile(`HTTPS_PROXY=(.*)\s*$`)
	match := regex.FindStringSubmatch(string(b))
	if len(match) > 1 {
		proxy.HttpsProxy = match[1]
	}

	regex = regexp.MustCompile(`HTTP_PROXY=(.*)\s*$`)
	match = regex.FindStringSubmatch(string(b))
	if len(match) > 1 {
		proxy.HttpProxy = match[1]
	}

	regex = regexp.MustCompile(`no_proxy=(.*)\s*$`)
	match = regex.FindStringSubmatch(string(b))
	if len(match) > 1 {
		proxy.NoProxy = match[1]
	}
	proxy.NoProxy = match[1]

	return proxy, nil
}

// startAdminPod starts an admin pod on a node so we can copy files.
func startAdminPod(restConfig *rest.Config, kubeClient kubernetes.Interface, kubeConfigPath string, nodeName string, registry string) (*kubectl.CopyConfig, error) {
	// first delete the pod in case there's an old one running
	k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, copyPodPrefix+"-"+nodeName)

	pod, err := k8s.StartAdminPodOnNode(kubeClient, nodeName, constants.OCNESystemNamespace, copyPodPrefix, false, registry)
	if err != nil {
		return nil, err
	}

	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, constants.OCNESystemNamespace, nil, false)
	if err != nil {
		return nil, err
	}

	return &kubectl.CopyConfig{KubectlConfig: kcConfig, PodName: pod.Name}, nil
}

// copyHAFilesToNode copies keepalived and nginx configuration files to the migrating node.
func copyHAFilesToNode(options *JoinOptions, bindPort uint16, altPort uint16, virtualIP string, proxy *types.Proxy, netInterface string) ([]string, error) {
	restConfig, kubeClient, err := client.GetKubeClient(options.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	cc, err := startAdminPod(restConfig, kubeClient, options.KubeConfigPath, options.Node, options.ClusterConfig.Registry)
	defer k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, copyPodPrefix+"-"+options.Node)
	if err != nil {
		return nil, err
	}

	// generate all of the files and systemd units needed to configure HA
	assets, err := ignition.GenerateAssetsForVirtualIp(bindPort, altPort, virtualIP, proxy, netInterface)
	if err != nil {
		return nil, err
	}

	// write each file to the node using "kubectl exec" on the admin pod
	for _, file := range assets.Files {
		if err := writeFileToNode(cc, file.Path, file.Mode, file.Contents.Source); err != nil {
			return nil, err
		}
	}

	// write each unit to the node using "kubectl exec" on the admin pod, including drop-ins
	enableServices := []string{}

	for _, unit := range assets.Units {
		path := "/etc/systemd/system/" + unit.Name

		if unit.Contents != nil && len(*unit.Contents) > 0 {
			if err := writeFileToNode(cc, path, 0755, *unit.Contents); err != nil {
				return nil, err
			}
		}
		if unit.Enabled != nil && *unit.Enabled {
			enableServices = append(enableServices, unit.Name)
		}

		for _, dropin := range unit.Dropins {
			d := path + ".d" + "/" + dropin.Name
			if err := writeFileToNode(cc, d, 0644, *dropin.Contents); err != nil {
				return nil, err
			}
		}
	}

	return enableServices, nil
}

// writeFileToNode uses "kubectl exec" to copy a file to a node.
func writeFileToNode(cc *kubectl.CopyConfig, path string, mode int, data string) error {
	remotePath := "/hostroot" + path
	dir := filepath.Dir(remotePath)

	// base64 encode the file contents so we don't have to deal with escaping characters
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	cmd := fmt.Sprintf(`mkdir -p %s && echo "%s" > /tmp/encoded && base64 -d /tmp/encoded > %s && chmod %o %s`, dir, encoded, remotePath, mode, remotePath)
	return kubectl.RunCommand(cc.KubectlConfig, cc.PodName, "sh", "-c", cmd)
}
