// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package byo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/certificate"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/cluster/kubepki"
	"github.com/oracle-cne/ocne/pkg/cluster/types"
	"github.com/oracle-cne/ocne/pkg/commands/image/create"
	conftypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	DriverName = "byo"
)

var DefaultProfile *conftypes.ByoProfile = nil

type ByoDriver struct {
	Name                 string
	Config               conftypes.ClusterConfig
	KubeconfigPath       string
	PKIInfo              *kubepki.PKIInfo
	UploadCertificateKey string
}

func CreateDriver(config *conftypes.Config, clusterConfig *conftypes.ClusterConfig) (driver.ClusterDriver, error) {

	kubeconfigPath, err := client.GetKubeconfigPath(fmt.Sprintf("kubeconfig.%s", clusterConfig.Name))
	if err != nil {
		return nil, err
	}

	uploadCertificateKey, err := util.CreateUploadCertificateKey()
	if err != nil {
		return nil, err
	}

	return &ByoDriver{
		Name:                 clusterConfig.Name,
		Config:               *clusterConfig,
		KubeconfigPath:       kubeconfigPath,
		UploadCertificateKey: uploadCertificateKey,
	}, nil
}

func (bd *ByoDriver) ignitionFromExtra(extraIgnition string, extraIgnitionInline string) (*igntypes.Config, error) {
	ret := &igntypes.Config{}
	var err error
	if extraIgnition != "" {
		ei := extraIgnition
		if !filepath.IsAbs(ei) {
			ei, err = filepath.Abs(filepath.Join(bd.Config.WorkingDirectory, ei))
			if err != nil {
				return nil, err
			}
		}
		ret, err = ignition.FromPath(ei)
		if err != nil {
			return nil, err
		}
	} else if extraIgnitionInline != "" {
		ret, err = ignition.FromString(extraIgnitionInline)
		if err != nil {
			return nil, err
		}
	}
	return ret, err
}

func (bd *ByoDriver) ignitionForNode(role types.NodeRole, join bool, joinToken string, caCertHashes []string, profile *conftypes.ByoProfile) ([]byte, error) {
	var ign *igntypes.Config
	var err error

	if profile == DefaultProfile {
		// The default profiles uses the primary extra ignition values.
		profile = &conftypes.ByoProfile{
			Name: "",
			ExtraIgnition: bd.Config.ExtraIgnition,
			ExtraIgnitionInline: bd.Config.ExtraIgnitionInline,
		}
	}

	// Overlay the extraIgnition if applicable

	internalLB := bd.Config.VirtualIp != ""
	kubeAPIServerIP := bd.getKubeAPIServerIP()

	// Make sure there is a network interface
	if bd.Config.Providers.Byo.NetworkInterface == "" {
		return nil, fmt.Errorf("A network interface must be provided")
	}

	if !join {
		// If a cluster is being initialized, then the CA certificate
		// and key need to be passed in to the new instance.
		caCert, err := os.ReadFile(bd.PKIInfo.CACertPath)
		if err != nil {
			return nil, err
		}
		//caKey, err = util.ToBase64(bd.PKIInfo.CAKeyPath)
		caKey, err := os.ReadFile(bd.PKIInfo.CAKeyPath)
		if err != nil {
			return nil, err
		}

		expectingWorkerNodes := bd.Config.WorkerNodes > 0
		ign, err = ignition.InitializeCluster(&ignition.ClusterInit{
			OsTag:                bd.Config.OsTag,
			OsRegistry:           bd.Config.OsRegistry,
			KubeAPIServerIP:      kubeAPIServerIP,
			KubeAPIBindPort:      bd.Config.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   bd.Config.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                bd.Config.Proxy,
			KubeAPIExtraSans:     []string{},
			KubePKICert:          string(caCert),
			KubePKIKey:           string(caKey),
			ServiceSubnet:        bd.Config.ServiceSubnet,
			PodSubnet:            bd.Config.PodSubnet,
			ExpectingWorkerNodes: expectingWorkerNodes,
			ProxyMode:            bd.Config.KubeProxyMode,
			ImageRegistry:        bd.Config.Registry,
			NetInterface:         bd.Config.Providers.Byo.NetworkInterface,
			UploadCertificateKey: bd.UploadCertificateKey,
			JoinToken:            joinToken,
			KubeVersion:          bd.Config.KubeVersion,
			TLSCipherSuites:      bd.Config.CipherSuites,
		})
	} else {
		// Worker nodes do not get two networks.  On remote clusters,
		// they only have a bridge network.  On local clusters, they
		// only have the user network.  The result is that they are
		// not impacted by the conflicting route problem that control
		// plane nodes suffer from.  Override the gateway so that
		// the real default route is not deleted.
		ign, err = ignition.JoinCluster(&ignition.ClusterJoin{
			Role:                 role,
			OsTag:                bd.Config.OsTag,
			OsRegistry:           bd.Config.OsRegistry,
			KubeAPIServerIP:      kubeAPIServerIP,
			JoinToken:            joinToken,
			KubePKICertHashes:    caCertHashes,
			ImageRegistry:        bd.Config.Registry,
			KubeAPIBindPort:      bd.Config.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   bd.Config.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                bd.Config.Proxy,
			ProxyMode:            bd.Config.KubeProxyMode,
			NetInterface:         bd.Config.Providers.Byo.NetworkInterface,
			UploadCertificateKey: bd.UploadCertificateKey,
			TLSCipherSuites:      bd.Config.CipherSuites,
		})
	}

	if err != nil {
		return nil, err
	}

	// Respect any proxy configuration that may be defined
	proxy, err := ignition.Proxy(&bd.Config.Proxy, kubeAPIServerIP, bd.Config.ServiceSubnet, bd.Config.PodSubnet)
	if err != nil {
		return nil, err
	}

	ign = ignition.Merge(ign, proxy)

	usrIgn, err := ignition.OcneUser(bd.Config.SshPublicKey, bd.Config.SshPublicKeyPath, bd.Config.Password)
	if err != nil {
		return nil, err
	}
	ign = ignition.Merge(ign, usrIgn)

	// If the ignition for the profile is to be overlaid on top
	// of the default profile, merge it in.
	if profile.Overlay {
		fromExtra, err := bd.ignitionFromExtra(bd.Config.ExtraIgnition, bd.Config.ExtraIgnitionInline)
		if err != nil {
			return nil, err
		}
		ign = ignition.Merge(ign, fromExtra)
	}

	fromExtra, err := bd.ignitionFromExtra(profile.ExtraIgnition, profile.ExtraIgnitionInline)
	if err != nil {
		return nil, err
	}
	ign = ignition.Merge(ign, fromExtra)

	return ignition.MarshalIgnition(ign)
}

func (bd *ByoDriver) printJoinCommands(joinToken string, uploadCerts bool) {
	// Print the commands to stderr so they are visible even if stdout
	// is redirected to a file.
	//
	// If a control plane node is being joined, print instructions
	// for uploading the certificate key as well.
	if uploadCerts {
		fmt.Fprintf(os.Stderr, "Run these commands before booting the new node to allow it to join the cluster:\n\t%s\n", k8s.UploadCertificateStanza(bd.UploadCertificateKey))
	} else {
		fmt.Fprintf(os.Stderr, "Run this command before booting the new node to allow it to join the cluster:\n")
	}

	fmt.Fprintf(os.Stderr, "\t%s\n", k8s.JoinTokenStanza(joinToken))
}

func (bd *ByoDriver) generateIso(doInit bool, caCertHashes []string, joinToken string) error {
	byoIgnitions := []*create.ByoIgnition{}
	tokenCreate := bd.Config.Providers.Byo.AutomaticTokenCreation

	// The init configuration always comes from the primary
	if doInit {
		jt := ""
		if tokenCreate {
			jt = joinToken
		}
		initIgn, err := bd.ignitionForNode(types.ControlPlaneRole, false, jt, caCertHashes, DefaultProfile)
		if err != nil {
			return err
		}

		byoIgnitions = append(byoIgnitions, &create.ByoIgnition{
			Name: "Initialize Cluster",
			Contents: initIgn,
		})

		// Can't create a token without one
		tokenCreate = false
	}

	joinIgn, err := bd.ignitionForNode(types.ControlPlaneRole, true, joinToken, caCertHashes, DefaultProfile)
	if err != nil {
		return err
	}
	byoIgnitions = append(byoIgnitions, &create.ByoIgnition{
		Name: "Control Plane Node",
		Contents: joinIgn,
	})

	// Generate a control plane and worker node configuration for each profile
	joinIgn, err = bd.ignitionForNode(types.WorkerRole, true, joinToken, caCertHashes, DefaultProfile)
	if err != nil {
		return err
	}
	byoIgnitions = append(byoIgnitions, &create.ByoIgnition{
		Name: "Worker Node",
		Contents: joinIgn,
	})


	// Generate the rest
	for _, profile := range bd.Config.Providers.Byo.Profiles {
		joinIgn, err = bd.ignitionForNode(types.ControlPlaneRole, true, joinToken, caCertHashes, profile)
		if err != nil {
			return err
		}
		byoIgnitions = append(byoIgnitions, &create.ByoIgnition{
			Name: fmt.Sprintf("%s - Control Plane Node", profile.Name),
			Contents: joinIgn,
		})

		joinIgn, err = bd.ignitionForNode(types.WorkerRole, true, joinToken, caCertHashes, profile)
		if err != nil {
			return err
		}

		byoIgnitions = append(byoIgnitions, &create.ByoIgnition{
			Name: fmt.Sprintf("%s - Worker Node", profile.Name),
			Contents: joinIgn,
		})
	}

	arch := runtime.GOARCH
	err = create.CreateIso(nil, &bd.Config, create.CreateOptions{
		ProviderType: create.ProviderTypeByo,
		Architecture: arch,
		Destination: fmt.Sprintf("%s-%s.iso", bd.Config.Name, arch),
		Byo: create.ByoOptions{
			Configurations: byoIgnitions,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (bd *ByoDriver) clusterInit() ([]byte, error) {
	// Generate the key material required for the cluster.  This includes
	// a PKI for the Kubernetes components as well as the admin kubeconfig.
	certOptions := certificate.CertOptions{
		Country: bd.Config.CertificateInformation.Country,
		Org:     bd.Config.CertificateInformation.Org,
		OrgUnit: bd.Config.CertificateInformation.OrgUnit,
		State:   bd.Config.CertificateInformation.State,
	}
	pkiInfo, err := kubepki.GeneratePKI(certOptions,
		kubepki.KubeconfigRequest{
			Path:           bd.KubeconfigPath,
			Host:           bd.getKubeAPIServerIP(),
			Port:           uint16(6443),
			ServiceSubnets: strings.Split(bd.Config.ServiceSubnet, ","),
		},
	)
	if err != nil {
		return nil, err
	}

	bd.PKIInfo = pkiInfo

	caCertHashes, err := k8s.CertHashesFromKubeconfig(bd.KubeconfigPath)

	if bd.Config.Providers.Byo.GenerateIso {
		// The join token needs to be embedded into the image.  The cluster
		// might not exist yet, so don't try to create the token.
		joinToken, err := k8s.CreateJoinToken(bd.GetKubeconfigPath(), true)
		if err != nil {
			return nil, err
		}

		err = bd.generateIso(true, caCertHashes, joinToken)
		if err != nil {
			return nil, err
		}

		// Print the join commands no matter what.  There's no telling when
		// this image might be used or re-used.  There are control plane
		// configurations in the image, so print the cert command.
		bd.printJoinCommands(joinToken, true)

		return nil, nil
	}

	initIgn, err := bd.ignitionForNode(types.ControlPlaneRole, false, "", caCertHashes, DefaultProfile)
	if err != nil {
		return nil, err
	}
	return initIgn, nil
}

func (bd *ByoDriver) Start() (bool, bool, error) {
	// Check for the Kubeconfig path.  If it exists, then say that
	// the cluster is already running
	log.Debugf("Checking for existing kubeconfig at %s", bd.KubeconfigPath)
	_, err := os.Stat(bd.KubeconfigPath)
	if err == nil {
		log.Debugf("Found existing kubeconfig")
		return true, false, nil
	}

	if err := bd.validateClusterConfig(); err != nil {
		return false, false, err
	}

	log.Debugf("Could not find existing kubeconfig.  Generating initialization materials")
	initIgn, err := bd.clusterInit()
	if err != nil {
		return false, false, err
	}

	// If an ISO was made, don't bother printing out the init ignition
	if !bd.Config.Providers.Byo.GenerateIso {
		fmt.Println(string(initIgn))
	}

	// Unlike providers that have infrastructure provisioning APIs, the BYO
	// provider just spits out some ignition.  It can take any amount of time
	// for the cluster to actually come up.  The typical use case will be to
	// take the ignition contents and write them to a file.  If that is what
	// is happening, then just exit after writing the string.  If instead
	// the call is attached to a TTY, prompt the user to continue once their
	// cluster node has started.
	isTTY, err := util.FileIsTTY(os.Stdout)
	if err != nil {
		return false, false, err
	}
	if !isTTY {
		return false, true, nil
	}

	for {
		var userInput string
		fmt.Println("When the first cluster node is initialized, press 'y' to continue the installation: ")
		fmt.Scanln(&userInput)
		if strings.EqualFold(userInput, "y") {
			break
		}
	}

	return false, false, nil
}

func (bd *ByoDriver) PostStart() error {
	// There is no post-start, so no-op
	return nil
}

func (bd *ByoDriver) Stop() error {
	return fmt.Errorf("The BYO provider does not support stopping a cluster")
}

func (bd *ByoDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	if err := bd.validateClusterConfig(); err != nil {
		return err
	}

	role := types.WorkerRole
	if controlPlaneNodes != 0 && workerNodes != 0 {
		return fmt.Errorf("The BYO provider cannot join worker and control plane nodes at the same time")
	} else if controlPlaneNodes != 0 {
		role = types.ControlPlaneRole
	}
	joinToken, err := k8s.CreateJoinToken(kubeconfigPath, !bd.Config.Providers.Byo.AutomaticTokenCreation)
	if err != nil {
		return err
	}

	log.Debugf("Got join token %s", joinToken)

	caCertHashes, err := k8s.CertHashesFromKubeconfig(kubeconfigPath)
	if err != nil {
		return err
	}
	log.Debugf("Cert hashes for %s are: %+v", kubeconfigPath, caCertHashes)

	doUploadCerts := (role == types.ControlPlaneRole)
	if bd.Config.Providers.Byo.GenerateIso {
		err = bd.generateIso(false, caCertHashes, joinToken)
		if err != nil {
			return err
		}

		// Generating the ISO creates profiles for both control
		// plane and worker nodes.  To account for this, always
		// do both the certificate upload and the join token stuff.
		doUploadCerts = true
	} else {
		ign, err := bd.ignitionForNode(role, true, joinToken, caCertHashes, DefaultProfile)
		if err != nil {
			return err
		}

		fmt.Println(string(ign))
	}

	if !bd.Config.Providers.Byo.AutomaticTokenCreation {
		bd.printJoinCommands(joinToken, doUploadCerts)
	} else if doUploadCerts {
		// If the expectation is that secrets are uploaded to the
		// cluster automatically and a control plane node is being
		// joined, then upload the certificates with the matching
		// key.
		err = k8s.UploadCertificates(bd.KubeconfigPath, bd.UploadCertificateKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bd *ByoDriver) Delete() error {
	// Remove the kubeconfigs
	log.Infof("Deleting file %s", bd.KubeconfigPath)
	err := os.Remove(bd.KubeconfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (bd *ByoDriver) Close() error {
	// Clean up the temp directory with the PKI files
	if bd.PKIInfo != nil && len(bd.PKIInfo.CertsDir) > 0 {
		os.RemoveAll(bd.PKIInfo.CertsDir)
	}
	return nil
}

func (bd *ByoDriver) GetKubeconfigPath() string {
	return bd.KubeconfigPath
}

func (bd *ByoDriver) GetKubeAPIServerAddress() string {
	return util.GetURIAddress(bd.getKubeAPIServerIP())
}

func (bd *ByoDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster:\n    use %s", bd.KubeconfigPath)
}

func (bd *ByoDriver) DefaultCNIInterfaces() []string {
	return []string{bd.Config.Providers.Byo.NetworkInterface}
}

func (bd *ByoDriver) getKubeAPIServerIP() string {
	if bd.Config.VirtualIp != "" {
		return bd.Config.VirtualIp
	}
	return bd.Config.LoadBalancer
}

func (bd *ByoDriver) validateClusterConfig() error {
	if bd.Config.VirtualIp == "" && bd.Config.LoadBalancer == "" {
		return fmt.Errorf("A virtual IP or load balancer is required")
	}

	if bd.Config.VirtualIp != "" && bd.Config.LoadBalancer != "" {
		return fmt.Errorf("Can not specify both virtual IP and load balancer")
	}

	return nil
}

// Stage is a no-op
func (bd *ByoDriver) Stage(version string) (string, string, bool, error) {
	return "", "", true, nil
}
