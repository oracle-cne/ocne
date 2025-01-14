// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/oracle-cne/ocne/pkg/certificate"
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/cluster/kubepki"
	"github.com/oracle-cne/ocne/pkg/cluster/types"
	conftypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/pidlock"
	log "github.com/sirupsen/logrus"
)

const (
	DriverName = "libvirt"

	// Manually specify the PCI bus/port for the bridge network
	// interface.  This ensures that the device name within Linux
	// can be calculated and fed into any configuration that gets
	// set within a VM.
	BridgeBus  = 1
	BridgeSlot = 0

	// Manually specify the PCI bus/port for user network interface
	// for the same reasons as the bridge network.
	UserBus  = 0
	UserSlot = 2

	// Calculate the bridge network device name from the bus/port
	// used to create it, following the Linux consistent network
	// device naming convention.  'en' stands for ethernet, 'p<num>'
	// stands for PCI port number (read: bus), 's<num>' stands for
	// the slot.
	BridgeNicPattern = "enp%ds%d"
)

// Volume defines a VM volume
type Volume struct {
	Key                string
	Name               string
	Path               string
	PathToBackingStore string
	Size               uint64
	StorageUnit        string
	Type               string
}

// PortForward defines a pair of ports.
// From is forwarded to To.  Listen is
// the IP to listen on.
type PortForward struct {
	From   uint16
	To     uint16
	Listen string
}

// Network defines a network for a VM
type Network struct {
	Type         string
	Network      string
	Bus          string
	Slot         string
	PortForwards []PortForward
}

// Domain defines a VM
type Domain struct {
	Name               string
	Description        string
	VolumePool         string
	Volume             string
	IgnitionPath       string
	Hypervisor         string
	Networks           []Network
	Memory             int
	MemoryCapacityUnit string
	CPUs               int
	CPUArch            string
}

// Pool defines a libvirt Pool
type Pool struct {
	Name string
	Path string
}

// LibvirtDriver manages resource creation and cluster management for libvirt
// targets.
type LibvirtDriver struct {
	Name          string // Name is the name of the cluster
	ClusterConfig conftypes.ClusterConfig

	// Basic connection information
	Connection *libvirt.Libvirt // Connection is the connection to libvirt
	URI        *url.URL         // URI is the URI of the libvirt target
	Local      bool             // Local indicates if the libvirt target is a local or remote system
	TargetIP   string           // TargetIP is the IP address of the libvirt target

	KubeAPIServerIP          string // KubeAPIServerIP is the IP address of Kubernetes from the perspective of the cluster nodes
	TunnelPort               uint16 // TunnelPort is a port on the target host that is used to tunnel connections to the Kubernetes API Server in the VM from the host
	LocalKubeconfigName      string
	LocalKubeconfigPath      string
	VMKubeconfigName         string
	VMKubeconfigPath         string
	PKIInfo                  *kubepki.PKIInfo
	NetworkName              string
	BridgeNetworking         bool
	NetworkingResolved       bool
	KubeVersion              string
	BootVolumeContainerImage string
	CPUArch                  string
	Info                     func(...interface{})
	Infof                    func(string, ...interface{})
	UploadCertificateKey     string
}

func CreateDriver(clusterConfig *conftypes.ClusterConfig) (driver.ClusterDriver, error) {
	lp := clusterConfig.Providers.Libvirt

	// Connect to the given libvirt URI.
	//
	// Note:  There is an issue in the libvirtd client on Mac when connecting
	//        to session URIs.  Specifically, the path to the socket it uses
	//        is incorrect.  Patch this up so that it is accurate for a default
	//        libvirt install.
	uri, err := url.Parse(*lp.SessionURI)
	if err != nil {
		return nil, err
	}

	// Different steps are required depending on whether the
	// target is local or remote.  For example, local VMs
	// require user networking with port forwarding while
	// remote
	hostIP, isLocal, err := util.ResolveURIToIP(uri)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" && uri.User.Username() == "" && uri.Host == "" && uri.Path != "" {
		// If a user gives an ssh-ish URI, something like user@host, then assume
		// they mean to use an SSH transport.  For whatever reason, when parsing
		// these URIs, url.Parse hands back a URL with no scheme, no user, and
		// no host.  However, the path field is populated with the connection
		// string.  This check may break in the future if the implementation of
		// url.Parse changes.
		*lp.SessionURI = fmt.Sprintf("qemu+ssh://%s/system", uri.Path)
		uri, err = url.Parse(*lp.SessionURI)
		hostIP, isLocal, err = util.ResolveURIToIP(uri)
		if err != nil {
			return nil, err
		}
	} else if isLocal && runtime.GOOS == "darwin" {
		// Mac handling.  Fix the URI and reparse.  There is
		// no need to call ResolveURIToIP again because the
		// address is still local
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		*lp.SessionURI = fmt.Sprintf("%s?socket=%s", *lp.SessionURI, filepath.Join(homedir, constants.DarwinLibvirtSocketPath))
		uri, err = url.Parse(*lp.SessionURI)
	}

	log.Debugf("Connecting to %s", uri.String())
	connection, err := libvirt.ConnectToURI(uri)
	if err != nil {
		return nil, err
	}

	localKubeconfigName := fmt.Sprintf("kubeconfig.%s.local", *clusterConfig.Name)
	vmKubeconfigName := fmt.Sprintf("kubeconfig.%s.vm", *clusterConfig.Name)

	vmKubeConfig, err := client.GetKubeconfigPath(vmKubeconfigName)
	if err != nil {
		return nil, err
	}

	localKubeConfig, err := client.GetKubeconfigPath(localKubeconfigName)
	if err != nil {
		return nil, err
	}

	architecture, err := getLibvirtCPUArchitecture(connection)
	if err != nil {
		return nil, err
	}
	info := log.Info
	infof := log.Infof
	if *clusterConfig.Quiet && log.GetLevel() != log.DebugLevel {
		info = func(a ...interface{}) {}
		infof = func(s string, a ...interface{}) {}
	}

	uploadCertificateKey, err := util.CreateUploadCertificateKey()
	if err != nil {
		return nil, err
	}

	ret := &LibvirtDriver{
		Name:                     *clusterConfig.Name,
		ClusterConfig:            *clusterConfig,
		Connection:               connection,
		URI:                      uri,
		Local:                    isLocal,
		TargetIP:                 hostIP,
		NetworkName:              *lp.Network,
		BridgeNetworking:         true,
		LocalKubeconfigName:      localKubeconfigName,
		LocalKubeconfigPath:      localKubeConfig,
		VMKubeconfigName:         vmKubeconfigName,
		VMKubeconfigPath:         vmKubeConfig,
		CPUArch:                  architecture,
		KubeVersion:              *clusterConfig.KubeVersion,
		BootVolumeContainerImage: *clusterConfig.BootVolumeContainerImage,
		Info:                     info,
		Infof:                    infof,
		UploadCertificateKey:     uploadCertificateKey,
	}

	return ret, nil
}

func (ld *LibvirtDriver) generateIgnition(nodeName string, role types.NodeRole, join bool, joinToken string, caCertHashes []string, bridgeNetwork bool, userNetwork bool) ([]byte, error) {
	var ign *igntypes.Config
	var err error

	var netInterface string
	if bridgeNetwork {
		netInterface = fmt.Sprintf(BridgeNicPattern, BridgeBus, BridgeSlot)
	} else {
		netInterface = fmt.Sprintf(BridgeNicPattern, UserBus, UserSlot)
	}

	internalLB := true
	if *ld.ClusterConfig.LoadBalancer != "" {
		internalLB = false
	}
	if ld.KubeAPIServerIP == "127.0.0.1" {
		internalLB = false
	}

	if !join {
		// If a cluster is being initialized, then the CA certificate
		// and key need to be passed in to the new instance.
		//caCert, err = util.ToBase64(ld.PKIInfo.CACertPath)
		var caCert []byte
		caCert, err = os.ReadFile(ld.PKIInfo.CACertPath)
		if err != nil {
			return nil, err
		}
		//caKey, err = util.ToBase64(ld.PKIInfo.CAKeyPath)
		var caKey []byte
		caKey, err = os.ReadFile(ld.PKIInfo.CAKeyPath)
		if err != nil {
			return nil, err
		}

		expectingWorkerNodes := *ld.ClusterConfig.WorkerNodes > 0
		ign, err = ignition.InitializeCluster(&ignition.ClusterInit{
			OsTag:                *ld.ClusterConfig.OsTag,
			OsRegistry:           *ld.ClusterConfig.OsRegistry,
			KubeAPIServerIP:      ld.KubeAPIServerIP,
			KubeAPIBindPort:      *ld.ClusterConfig.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   *ld.ClusterConfig.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                ld.ClusterConfig.Proxy,
			KubeAPIExtraSans:     []string{ld.TargetIP},
			KubePKICert:          string(caCert),
			KubePKIKey:           string(caKey),
			ServiceSubnet:        *ld.ClusterConfig.ServiceSubnet,
			PodSubnet:            *ld.ClusterConfig.PodSubnet,
			ExpectingWorkerNodes: expectingWorkerNodes,
			ProxyMode:            *ld.ClusterConfig.KubeProxyMode,
			ImageRegistry:        *ld.ClusterConfig.Registry,
			NetInterface:         netInterface,
			UploadCertificateKey: ld.UploadCertificateKey,
			KubeVersion:          ld.KubeVersion,
			TLSCipherSuites:      *ld.ClusterConfig.CipherSuites,
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
			OsTag:                *ld.ClusterConfig.OsTag,
			OsRegistry:           *ld.ClusterConfig.OsRegistry,
			KubeAPIServerIP:      ld.KubeAPIServerIP,
			JoinToken:            joinToken,
			KubePKICertHashes:    caCertHashes,
			ImageRegistry:        *ld.ClusterConfig.Registry,
			KubeAPIBindPort:      *ld.ClusterConfig.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   *ld.ClusterConfig.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                ld.ClusterConfig.Proxy,
			ProxyMode:            *ld.ClusterConfig.KubeProxyMode,
			NetInterface:         netInterface,
			UploadCertificateKey: ld.UploadCertificateKey,
			TLSCipherSuites:      *ld.ClusterConfig.CipherSuites,
		})
	}

	if err != nil {
		return nil, err
	}

	// If there is a bridge network, create the configuration file.
	// Whether or not this interface hosts the default route depends
	// on whether or not there is a user interface as well.  If there
	// is, then the bridge interface is not the default route.  If not,
	// then it is.
	if bridgeNetwork {
		bridgeNet := ignition.DefaultNetwork()
		bridgeNet.Name = fmt.Sprintf(BridgeNicPattern, BridgeBus, BridgeSlot)
		bridgeNet.DefaultRoute = !userNetwork
		bridgeNet.IPV6DefaultRoute = bridgeNet.DefaultRoute

		bridgeFile, err := bridgeNet.ToFile()
		if err != nil {
			return nil, err
		}
		ignition.AddFile(ign, bridgeFile)
	}

	// If there is a user network, add the configuration file.
	if userNetwork {
		userNet := ignition.DefaultNetwork()
		userNet.Name = fmt.Sprintf(BridgeNicPattern, UserBus, UserSlot)

		userFile, err := userNet.ToFile()
		if err != nil {
			return nil, err
		}
		ignition.AddFile(ign, userFile)
	}

	// Respect any proxy configuration that may be defined
	proxy, err := ignition.Proxy(&ld.ClusterConfig.Proxy, ld.KubeAPIServerIP, *ld.ClusterConfig.ServiceSubnet, *ld.ClusterConfig.PodSubnet)
	if err != nil {
		return nil, err
	}

	ign = ignition.Merge(ign, proxy)

	// For libvirt add hostname to ignition
	hostnameFile := ignition.File{
		Path: "/etc/hostname",
		Mode: 0644,
		Contents: ignition.FileContents{
			Source: nodeName,
		},
	}
	ignition.AddFile(ign, &hostnameFile)

	usrIgn, err := ignition.OcneUser(*ld.ClusterConfig.SshPublicKey, *ld.ClusterConfig.SshPublicKeyPath, *ld.ClusterConfig.Password)
	if err != nil {
		return nil, err
	}
	ign = ignition.Merge(ign, usrIgn)

	// Add any additional configuration
	if *ld.ClusterConfig.ExtraIgnition != "" {
		ei := *ld.ClusterConfig.ExtraIgnition
		if !filepath.IsAbs(ei) {
			ei, err = filepath.Abs(filepath.Join(*ld.ClusterConfig.WorkingDirectory, ei))
			if err != nil {
				return nil, err
			}
		}
		fromExtra, err := ignition.FromPath(ei)
		if err != nil {
			return nil, err
		}
		ign = ignition.Merge(ign, fromExtra)
	}
	if *ld.ClusterConfig.ExtraIgnitionInline != "" {
		fromExtra, err := ignition.FromString(*ld.ClusterConfig.ExtraIgnitionInline)
		if err != nil {
			return nil, err
		}
		ign = ignition.Merge(ign, fromExtra)
	}

	return ignition.MarshalIgnition(ign)
}

func (ld *LibvirtDriver) addNode(role types.NodeRole, num int, join bool, tokenStr string, caCertHashes []string) error {
	libvirtDomainName := getDomainName(ld.Name, role, num)
	libvirtVolumeName, ignitionVolumeName := getResourceNames(libvirtDomainName)
	userNetworking := true
	if join {
		log.Debugf("Adding node %s to the cluster", libvirtDomainName)
		userNetworking = false
	} else {
		log.Debugf("Initializing a cluster with first node %s", libvirtDomainName)
	}

	log.Debugf("Generating Ignition file")
	ignitionBytes, err := ld.generateIgnition(libvirtDomainName, role, join, tokenStr, caCertHashes, ld.BridgeNetworking, userNetworking)
	if err != nil {
		return err
	}

	// Ensure all necessary libvirt infrastructure exists
	log.Debugf("Ensuring presence of storage pool")
	imagesPool, err := FindOrCreateStoragePool(ld.Connection, ld.URI, *ld.ClusterConfig.Providers.Libvirt.StoragePool)
	if err != nil {
		return err
	}

	// If the volume of the base image does not exist on the system, it is not downloaded
	log.Debugf("Checking if base image exists")

	bootVolumeName := *ld.ClusterConfig.Providers.Libvirt.BootVolumeName
	if bootVolumeName == constants.BootVolumeName {
		// Boot volume name has not been overridden, add the k8s version to the name
		bootVolumeName = bootVolumeName + "-" + ld.KubeVersion
	}

	// The storage pool is refreshed
	log.Debugf("Refreshing storage pools to determine whether boot volume file exists or not in the pool")
	if err = refreshStoragePools(ld.Connection); err != nil {
		return err
	}

	if !DoesBaseImageExist(ld.Connection, imagesPool, bootVolumeName) {
		//The base image is downloaded and placed into the pool named images
		if err = TransferBaseImage(ld.Connection, ld.BootVolumeContainerImage, imagesPool, bootVolumeName, ld.CPUArch); err != nil {
			return err
		}
	}

	// Make a new volume if necessary.
	log.Debugf("Checking if volume %s exists", libvirtVolumeName)
	_, err = ld.Connection.StorageVolLookupByName(*imagesPool, libvirtVolumeName)
	if checkLibvirtError(err, libvirt.ErrNoStorageVol) {
		// Make a volume
		log.Debugf("Creating volume %s", libvirtVolumeName)
		if role == types.ControlPlaneRole {
			_, err = createVolumeFromImagesPool(ld.ClusterConfig.Providers.Libvirt.ControlPlaneNode, ld.Connection, imagesPool, libvirtVolumeName, bootVolumeName)
		} else {
			_, err = createVolumeFromImagesPool(ld.ClusterConfig.Providers.Libvirt.WorkerNode, ld.Connection, imagesPool, libvirtVolumeName, bootVolumeName)
		}
	}
	if err != nil {
		return err
	}

	log.Debugf("Uploading ignition file to %s", ignitionVolumeName)
	if err = uploadInitialIgnitionFile(ld.Connection, ignitionBytes, imagesPool, ignitionVolumeName); err != nil {
		return err
	}

	// The storage pool is refreshed
	log.Debugf("Refreshing storage pools")
	if err = refreshStoragePools(ld.Connection); err != nil {
		return err
	}

	// A new domain/VM is spun up on the OL8 instance, using the volume that has just been created
	ignitionPath, err := getVolumePath(ld.Connection, imagesPool, ignitionVolumeName)
	if err != nil {
		return err
	}

	// Assume that the target system is running Linux, and
	// by extension that the default hypervisor is KVM.
	// If the target is local and is a mac, then assume
	// HVF
	hypervisor := "kvm"
	if ld.Local && runtime.GOOS == "darwin" {
		hypervisor = "hvf"
	}

	domNets := []Network{}

	// All control plane nodes need a user network.
	// Remote clusters need one to enable port forwarding
	// to kube-apiserver from the host without having to
	// do any non-libvirt configuration.  Local clusters
	// need one because normal users are not privileged to
	// create VNICs.
	if userNetworking {
		userNetwork := Network{
			Type: "user",
			Bus:  fmt.Sprintf("%d", UserBus),
			Slot: fmt.Sprintf("%d", UserSlot),
		}
		if *ld.ClusterConfig.LoadBalancer == "" {
			userNetwork.PortForwards = []PortForward{
				{
					From:   ld.TunnelPort,
					To:     6443,
					Listen: ld.TargetIP,
				},
			}
		}
		domNets = append(domNets, userNetwork)
	}

	// If bridge networking is desired, add the network
	if ld.BridgeNetworking {
		domNets = append(domNets, Network{
			Type:    "network",
			Network: ld.NetworkName,
			Bus:     fmt.Sprintf("0x%x", BridgeBus),
			Slot:    fmt.Sprintf("0x%x", BridgeSlot),
		})
	}

	if len(domNets) == 0 {
		return fmt.Errorf("No networks defined for node")
	}

	log.Debugf("Creating domain %s", libvirtDomainName)
	domainInformation := Domain{
		Name:         libvirtDomainName,
		Description:  "An instance that is spun up dynamically",
		VolumePool:   imagesPool.Name,
		Volume:       libvirtVolumeName,
		IgnitionPath: ignitionPath,
		Hypervisor:   hypervisor,
		Networks:     domNets,
	}
	if role == types.ControlPlaneRole {
		err = createDomainFromTemplate(ld.ClusterConfig.Providers.Libvirt.ControlPlaneNode, ld.Connection, &domainInformation)
	} else {
		err = createDomainFromTemplate(ld.ClusterConfig.Providers.Libvirt.WorkerNode, ld.Connection, &domainInformation)
	}
	if err != nil {
		return err
	}
	return nil
}

func (ld *LibvirtDriver) removeNode(libvirtDomainName string) error {
	libvirtVolumeName, ignitionVolumeName := getResourceNames(libvirtDomainName)

	// Stop and undefine the domain
	err := removeDomain(ld.Connection, libvirtDomainName, ld.CPUArch)
	if err != nil {
		return err
	}

	poolPath, _, err := getDefaultPoolPath(ld.URI)
	if err != nil {
		return err
	}

	imagesPool, err := FindStoragePool(ld.Connection, *ld.ClusterConfig.Providers.Libvirt.StoragePool, poolPath)
	if err != nil {
		return err
	} else if imagesPool == nil {
		// The goal is to ensure images don't exist in a pool.
		// If the pool does not exist, then the images cannot
		// be in it.
		ld.Infof("Could not find pool with path %s", poolPath)
		return nil
	}

	// Delete the ignition volume
	ld.Infof("Deleting volume %s", ignitionVolumeName)
	err = deleteVolume(ld.Connection, imagesPool, ignitionVolumeName)
	if err != nil {
		return err
	}

	// Delete the boot volume
	ld.Infof("Deleting volume %s", libvirtVolumeName)
	err = deleteVolume(ld.Connection, imagesPool, libvirtVolumeName)
	if err != nil {
		return err
	}

	return nil
}

func (ld *LibvirtDriver) resolveNetworking() error {
	if ld.NetworkingResolved {
		return nil
	}
	ld.NetworkingResolved = true

	// Check to see if the desired network is available and if any networks
	// exist at all.  If the desired network exists, the use it.  If no
	// networks exist, then assume that user-only networking is the right
	// thing to do.  If there are networks defined, but not the configured
	// one, then hand back an error.
	netExists, netsExist, err := doesNetworkExist(ld.Connection, ld.NetworkName)
	if err != nil {
		return err
	} else if !netsExist {
		// If there are no networks, then only use user networks.
		ld.BridgeNetworking = false
		return nil
	} else if !netExists {
		// If there are networks, but not the one that was asked
		// for, complain about it.
		return fmt.Errorf("Network %s does not exist", ld.NetworkName)
	}

	// The desired network exists.  Bridge networking is usable, so enable it.
	ld.BridgeNetworking = true
	return nil
}

func (ld *LibvirtDriver) initializeCluster() error {
	// allocate a port that the VM binds to on the target host to
	// allow access to the kube-apiserver.
	err := pidlock.WaitFor(10 * time.Second)
	if err != nil {
		return err
	}
	defer pidlock.Drop()

	port, err := allocatePort(ld.URI.Hostname())
	if err != nil {
		return err
	}
	ld.TunnelPort = port
	log.Debugf("Tunnel port is %d", ld.TunnelPort)

	// Generate the certificates and kubeconfigs required to instantiate
	// and use the cluster.  Two kubeconfigs are created.  One is the
	// kubeconfig that the references the canonical address of the cluster
	// API server.  The other is one that talks to the user-network tunnel
	// between the target host and the VM itself.
	certOptions := certificate.CertOptions{
		Country: *ld.ClusterConfig.CertificateInformation.Country,
		Org:     *ld.ClusterConfig.CertificateInformation.Org,
		OrgUnit: *ld.ClusterConfig.CertificateInformation.OrgUnit,
		State:   *ld.ClusterConfig.CertificateInformation.State,
	}
	var krLocal kubepki.KubeconfigRequest
	if *ld.ClusterConfig.LoadBalancer == "" {
		krLocal = kubepki.KubeconfigRequest{
			Path:          ld.LocalKubeconfigPath,
			Host:          ld.TargetIP,
			Port:          port,
			ServiceSubnet: *ld.ClusterConfig.ServiceSubnet,
		}
	} else {
		krLocal = kubepki.KubeconfigRequest{
			Path:          ld.LocalKubeconfigPath,
			Host:          *ld.ClusterConfig.LoadBalancer,
			Port:          uint16(6443),
			ServiceSubnet: *ld.ClusterConfig.ServiceSubnet,
		}
	}
	pkiInfo, err := kubepki.GeneratePKI(certOptions,
		kubepki.KubeconfigRequest{
			Path:          ld.VMKubeconfigPath,
			Host:          ld.KubeAPIServerIP,
			Port:          uint16(6443),
			ServiceSubnet: *ld.ClusterConfig.ServiceSubnet,
		},
		krLocal,
	)
	if err != nil {
		return err
	}

	ld.PKIInfo = pkiInfo

	// Initialize the cluster by adding the first node.
	err = ld.addNode(types.ControlPlaneRole, 1, false, "", []string{""})
	if err != nil {
		return err
	}

	addCluster(ld.URI.Hostname(), ld.Name, ld.KubeAPIServerIP, port)
	return nil
}

func (ld *LibvirtDriver) create() error {
	ld.Infof("Creating new Kubernetes cluster with version %s named %s", ld.KubeVersion, ld.Name)

	var err error
	kubeAPIServerIP := "127.0.0.1"
	if ld.BridgeNetworking {
		// use VirtualIp as KubeAPIServerIP if VirtualIp is specified
		// use LoadBalancer as KubeAPIServerIP if LoadBalancer is specified
		// otherwise, allocate an IP that will be used for the VM if a remote
		if *ld.ClusterConfig.VirtualIp != "" {
			kubeAPIServerIP = *ld.ClusterConfig.VirtualIp
		} else if *ld.ClusterConfig.LoadBalancer != "" {
			kubeAPIServerIP = *ld.ClusterConfig.LoadBalancer
		} else {
			kubeAPIServerIP, err = allocateIP(ld.Connection, ld.URI.Hostname(), ld.NetworkName)
			if err != nil {
				return err
			}
		}
	}
	ld.KubeAPIServerIP = kubeAPIServerIP
	log.Debugf("Kubernetes API Server address is %s", ld.KubeAPIServerIP)

	err = ld.initializeCluster()
	if err != nil {
		return err
	}

	// Wait for the cluster to become responsive
	_, kubeClient, err := client.GetKubeClient(ld.LocalKubeconfigPath)
	if err != nil {
		return err
	}

	_, err = k8s.WaitUntilGetNodesSucceeds(kubeClient)
	if err != nil {
		return err
	}

	// Add in other control plane nodes
	if *ld.ClusterConfig.ControlPlaneNodes > 1 {
		err = ld.join(types.ControlPlaneRole, *ld.ClusterConfig.ControlPlaneNodes)
		if err != nil {
			return err
		}
	}

	// Add in worker nodes
	err = ld.join(types.WorkerRole, *ld.ClusterConfig.WorkerNodes)
	if err != nil {
		return err
	}

	return nil
}

func (ld *LibvirtDriver) join(role types.NodeRole, num uint16) error {
	// Make sure there is more than zero nodes to add
	if num <= 0 {
		return nil
	}

	err := ld.resolveNetworking()
	if err != nil {
		return err
	}

	if !ld.BridgeNetworking {
		return fmt.Errorf("Adding nodes to user-only network clusters is not supported")
	}

	// Get join configuration for new nodes.  The nodes
	// are joining from the perspective of the VM, so use
	// that kubeconfig to for the CA materials.  The token
	// has to be pushed to the cluster, which is accessed
	// via the local kubeconfig.  So use that one there.
	caCertHashes, err := k8s.CertHashesFromKubeconfig(ld.VMKubeconfigPath)
	if err != nil {
		return err
	}
	tokenStr, err := k8s.CreateJoinToken(ld.LocalKubeconfigPath, false)
	if err != nil {
		return nil
	}
	startNum := 1
	// for control planes nodes, it starts at 2
	if role == types.ControlPlaneRole {
		startNum = 2
	}
	var i uint16
	for i = uint16(startNum); i <= num; i++ {
		err := ld.addNode(role, int(i), true, tokenStr, caCertHashes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ld *LibvirtDriver) Start() (bool, bool, error) {
	// Check the cache for a cluster of the same name.
	clusterCache, err := cache.GetCache()
	if err != nil {
		return false, false, err
	}

	existingClusterConfig := clusterCache.Get(*ld.ClusterConfig.Name)
	if existingClusterConfig != nil {
		// Make sure the sessions are the same
		if *ld.ClusterConfig.Providers.Libvirt.SessionURI != *existingClusterConfig.ClusterConfig.Providers.Libvirt.SessionURI {
			return false, false, fmt.Errorf("A cluster named %s already exists for the libvirt provider but has session URI %s", *ld.ClusterConfig.Name, *existingClusterConfig.ClusterConfig.Providers.Libvirt.SessionURI)
		}
	}

	err = ld.resolveNetworking()
	if err != nil {
		return false, false, err
	}

	// Do some quick validation.
	if !ld.BridgeNetworking && *ld.ClusterConfig.WorkerNodes > 0 {
		return false, false, fmt.Errorf("Adding worker nodes to user-networking clusters is not supported")
	}
	if !ld.BridgeNetworking && *ld.ClusterConfig.ControlPlaneNodes > 1 {
		return false, false, fmt.Errorf("Adding more than one control plane nodes to user-networking clusters is not supported")
	}

	if *ld.ClusterConfig.VirtualIp != "" && *ld.ClusterConfig.LoadBalancer != "" {
		return false, false, fmt.Errorf("Can not specify both virtual IP and load balancer")
	}

	// Before making anything new, check if the domain that is about to be
	// created already exists.  If so, don't bother trying to make any of
	// this stuff.  If it exists and not started, try to start it.
	firstControlPlaneNode := getDomainName(ld.Name, types.ControlPlaneRole, 1)
	isRunning, wasRunning, err := setDomainRunningIfExists(ld.Connection, firstControlPlaneNode)
	if isRunning {
		isRunning, err = isClusterUp(ld.LocalKubeconfigPath, !wasRunning)
		if isRunning {
			ld.Infof("Cluster %s is running already", ld.Name)
		} else {
			log.Errorf("Failed to start cluster %q. The libvirt domain %q is running.", ld.Name, firstControlPlaneNode)
		}
		return isRunning, false, err
	} else if err != nil {
		return false, false, err
	}

	// If the cluster did not already exist, create it
	return false, false, ld.create()
}

func (ld *LibvirtDriver) PostStart() error {
	return nil
}

func (ld *LibvirtDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	return fmt.Errorf("Join not implemented for libvirt provider")
}

func (ld *LibvirtDriver) Stop() error {
	return nil
}

func (ld *LibvirtDriver) Delete() error {
	// Iterate over all domains and find domains that match what are
	// expected for this cluster.
	flags := libvirt.ConnectListDomainsActive | libvirt.ConnectListDomainsInactive
	doms, _, err := ld.Connection.ConnectListAllDomains(1, flags)
	if err != nil {
		return err
	}

	for _, d := range doms {
		// Only delete stuff that shares a cluster name
		if !isDomainFromCluster(d.Name, ld.Name) {
			continue
		}

		err = ld.removeNode(d.Name)
		if err != nil {
			return err
		}
	}

	// Remove the kubeconfigs
	ld.Infof("Deleting file %s", ld.LocalKubeconfigPath)
	err = os.Remove(ld.LocalKubeconfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	ld.Infof("Deleting file %s", ld.VMKubeconfigPath)
	err = os.Remove(ld.VMKubeconfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete the cluster network data from the cache
	err = pidlock.WaitFor(10 * time.Second)
	if err != nil {
		return err
	}
	removeCluster(ld.Name)
	pidlock.Drop()

	return nil
}

func (ld *LibvirtDriver) Close() error {
	// Clean up the temp directory with the PKI files
	if ld.PKIInfo != nil && len(ld.PKIInfo.CertsDir) > 0 {
		os.RemoveAll(ld.PKIInfo.CertsDir)
	}
	return ld.Connection.Disconnect()
}

func (ld *LibvirtDriver) GetKubeconfigPath() string {
	return ld.LocalKubeconfigPath
}

func (ld *LibvirtDriver) GetKubeAPIServerAddress() string {
	return ld.TargetIP
}

func (ld *LibvirtDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster from the VM host:\n    copy %s to that host and run kubectl there\nTo access the cluster from this system:\n    use %s", ld.VMKubeconfigPath, ld.LocalKubeconfigPath)
}

func (ld *LibvirtDriver) DefaultCNIInterfaces() []string {
	if ld.BridgeNetworking {
		return []string{fmt.Sprintf(BridgeNicPattern, BridgeBus, BridgeSlot)}
	}
	return []string{""}
}

// Stage is a no-op
func (ld *LibvirtDriver) Stage(version string) (string, string, bool, error) {
	return "", "", true, nil
}
