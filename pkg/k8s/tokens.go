// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"crypto/x509"
	"fmt"

	"github.com/oracle-cne/ocne/pkg/k8s/client"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	tokens "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
	kcfgutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
)

// CertsFromKubeconfig gets the list of x509 certificates embedded in
// a kubeconfig.
func CertsFromKubeconfig(kubeconfigPath string) ([]*x509.Certificate, error) {
	conf, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	_, clusterConf := kcfgutil.GetClusterFromKubeConfig(conf)

	if clusterConf.CertificateAuthorityData != nil {
		return cert.ParseCertsPEM(clusterConf.CertificateAuthorityData)
	} else {
		return cert.CertsFromFile(clusterConf.CertificateAuthority)
	}

	return nil, fmt.Errorf("Kubeconfig did not have any CAs")
}

// CertHashesFromKubeconfig returns a list of hashes, one for each x509
// certificate in a kubeconfig
func CertHashesFromKubeconfig(kubeconfigPath string) ([]string, error) {
	certs, err := CertsFromKubeconfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, c := range certs {
		ret = append(ret, pubkeypin.Hash(c))
	}

	return ret, nil
}

func CreateJoinToken(kubeconfigPath string, generateOnly bool) (string, error) {
	// Create the token
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", err
	}

	if generateOnly {
		return tokenStr, nil
	}

	bsTokenString, err := bootstraptokenv1.NewBootstrapTokenString(tokenStr)
	if err != nil {
		return "", err
	}

	bsToken := bootstraptokenv1.BootstrapToken{}
	bootstraptokenv1.SetDefaults_BootstrapToken(&bsToken)
	bsToken.Token = bsTokenString

	clientset, err := client.GetKubernetesClientset(kubeconfigPath)
	if err != nil {
		return "", err
	}

	err = tokens.CreateNewTokens(clientset, []bootstraptokenv1.BootstrapToken{bsToken})
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// CreateJoin creates a join token on an existing cluster and returns the token
// as well as the hashes of the CA certs for the input kubeconfig
func CreateJoin(kubeconfigPath string) (string, []string, error) {

	// Extract the CA certificate hashes from the kubeconfig
	// Please refer to https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/cmd/util/join.go#L52
	// for details.
	caCertHashes, err := CertHashesFromKubeconfig(kubeconfigPath)
	if err != nil {
		return "", nil, err
	}

	// Once all the necessary values have been generated, push
	// the token to the cluster.
	tokenStr, err := CreateJoinToken(kubeconfigPath, false)
	if err != nil {
		return "", nil, err
	}

	return tokenStr, caCertHashes, nil
}

// UploadCertificateStanza returns a command to run that will upload
// certificates to the cluster to enable joining new control plane nodes.
func UploadCertificateStanza(kubeconfigPath string, key string) (string, error) {
	clientset, err := client.GetKubernetesClientset(kubeconfigPath)
	if err != nil {
		return "", err
	}

	// Get a list of control plane nodes
	nodes, err := GetControlPlaneNodes(clientset)
	if err != nil {
		return "", err
	}

	// There needs to be at least one.  There always should be.
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("Did not find any control plane nodes")
	}

	// Choose the first node in the list.
	nodeName := nodes.Items[0].ObjectMeta.Name
	return fmt.Sprintf("echo \"chroot /hostroot kubeadm init phase upload-certs --certificate-key %s --upload-certs\" | ocne cluster console --node %s", key, nodeName), nil
}

// UploadCertificates uploads the key material required for control
// plane nodes to join clusters.
func UploadCertificates(kubeconfigPath string, key string) error {
	restConfig, client, err := client.GetKubeClient(kubeconfigPath)
	if err != nil {
		return err
	}

	// The certificate upload has to happen on a control plane node
	// that is already part of the cluster.  If not, the upload-certs
	// command will generate it's own certificates and upload those.
	// That's not going to work because it will use a new root CA.
	// So rather than a nice, simple API call it is necessary to reach
	// out to a node and run a command on it.
	nodes, err := GetControlPlaneNodes(client)
	if err != nil {
		return err
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("Did not find any control plane nodes")
	}

	nodeName := nodes.Items[0].ObjectMeta.Name

	err = CreateNamespaceIfNotExists(client, constants.OCNESystemNamespace)
	if err != nil {
		return err
	}

	pod, err := StartAdminPodOnNode(client, nodeName, constants.OCNESystemNamespace, "exec", false)
	if err != nil {
		return err
	}
	defer DeletePod(client, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	ignore := []string{
		"could not fetch a Kubernetes version",
		"falling back to",
	}
	kc, err := kubectl.NewKubectlConfig(restConfig, kubeconfigPath, pod.ObjectMeta.Namespace, ignore, true)
	if err != nil {
		return err
	}

	return kubectl.RunCommand(kc, pod.ObjectMeta.Name,
		"chroot",
		"/hostroot",
		"kubeadm",
		"init",
		"phase",
		"upload-certs",
		"--certificate-key",
		key,
		"--upload-certs",
	)
}
