// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"encoding/xml"
	"fmt"
	"github.com/digitalocean/go-libvirt"
	"github.com/korylprince/ipnetgen"
	"gopkg.in/yaml.v3"
	"github.com/oracle-cne/ocne/pkg/constants"
	"net"
	"os"
	"path/filepath"
)

type LibvirtNetwork struct {
	IP struct {
		Address string `xml:"address,attr"`
		Netmask string `xml:"netmask,attr"`
	} `xml:"ip"`
}

type UsedIPs map[string]bool
type UsedPorts map[uint16]bool

type HostData struct {
	IPs   UsedIPs
	Ports UsedPorts
}

type ClusterData struct {
	Host string
	IP   string
	Port uint16
}

type NetworkInformation struct {
	Hosts    map[string]*HostData
	Clusters map[string]*ClusterData
}

func getUsedAddrFilePath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homedir, constants.UserConfigDir, constants.UserIPData), nil
}

func getHostData(ni *NetworkInformation, host string) *HostData {
	ret, ok := ni.Hosts[host]
	if !ok {
		ret = &HostData{
			IPs:   UsedIPs{},
			Ports: UsedPorts{},
		}
		ni.Hosts[host] = ret
	}

	// Protect against problematic edits to
	// the file.
	if ret.IPs == nil {
		ret.IPs = UsedIPs{}
	}
	if ret.Ports == nil {
		ret.Ports = UsedPorts{}
	}
	return ret
}

func getNetworkInformation() (NetworkInformation, error) {
	ret := NetworkInformation{
		Hosts:    map[string]*HostData{},
		Clusters: map[string]*ClusterData{},
	}

	addrPath, err := getUsedAddrFilePath()
	if err != nil {
		return ret, err
	}

	yamlBytes, err := os.ReadFile(addrPath)
	if err != nil {
		// If the file does not exist, then ignore
		// the error.  It will be initialized later
		// on and saved if appropriate.  Treat a
		// missing file as one that has no
		// used addresses.
		if os.IsNotExist(err) {
			return ret, nil
		}
		return ret, err
	}

	err = yaml.Unmarshal(yamlBytes, &ret)
	return ret, err
}

func saveNetworkInformation(ni NetworkInformation) error {
	addrPath, err := getUsedAddrFilePath()
	if err != nil {
		return err
	}

	outBytes, err := yaml.Marshal(&ni)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(addrPath), 0700)
	if err != nil {
		return err
	}
	return os.WriteFile(addrPath, outBytes, 0600)
}

// doesNetworkExist checks to see if a network with a given name is defined.
// If not, it will check to see if *any* networks are defined.  The first
// return value indicates if the named network exists.  The second indicates
// if any networks are defined.
func doesNetworkExist(l *libvirt.Libvirt, networkName string) (bool, bool, error) {
	_, err := l.NetworkLookupByName(networkName)

	// If there is no error, then the network exists.  If one network exists,
	// then it cannot be true that no networks exist.  Therefore, both return
	// values must be true.
	if err == nil {
		return true, true, nil
	}

	if !checkLibvirtError(err, libvirt.ErrNoNetwork) {
		return false, false, err
	}

	// The named network did not exist.  Time to check if there are
	// any networks at all.
	netNames, err := l.ConnectListNetworks(0)
	if err != nil {
		return false, false, err
	}

	return false, len(netNames) > 0, nil
}

func getNetworkDetails(l *libvirt.Libvirt, networkName string) (*LibvirtNetwork, error) {
	// Find the network configuration for the "default" network
	// and then parse out the information that defines its subnet
	libvirtNet, err := l.NetworkLookupByName(networkName)
	if err != nil {
		return nil, err
	}

	xmlDesc, err := l.NetworkGetXMLDesc(libvirtNet, 0)
	if err != nil {
		return nil, err
	}

	decoded := LibvirtNetwork{}

	err = xml.Unmarshal([]byte(xmlDesc), &decoded)
	if err != nil {
		return nil, err
	}
	return &decoded, nil
}

func allocateIP(l *libvirt.Libvirt, hostname string, networkName string) (string, error) {
	decoded, err := getNetworkDetails(l, networkName)
	if err != nil {
		return "", err
	}

	// At this point, the subnet is known.  It is now
	// possible to pick an address near the top of the range
	// and use that.
	//
	// Go does not have an efficient way to work with subnet
	// masks.  It appears the easiest approach is to convert
	// the mask to an IP, and then cast the byte array to
	// an IPMask.
	//
	// TODO: look at the DHCP configuration and see if there
	//       is an unused block.  If so, use that bit.
	subnetIp := net.ParseIP(decoded.IP.Address)
	subnetMaskIp := net.ParseIP(decoded.IP.Netmask)
	subnetMask := net.IPMask(subnetMaskIp)

	// Libvirt hands back the gateway ip and a subnet mask
	// This must be converted into a subnet.  Mask off
	// individual bytes of the address.  It should not be
	// possible to have the lengths of these two things be
	// different, but it's a good idea to check anyway.
	if len(subnetIp) != len(subnetMask) {
		return "", fmt.Errorf("The gateway address %s is not the same length as the subnet mask %s", decoded.IP.Address, decoded.IP.Netmask)
	}
	for i := 0; i < len(subnetIp); i++ {
		subnetIp[i] = subnetIp[i] & subnetMask[i]
	}

	subnet := net.IPNet{
		IP:   subnetIp,
		Mask: subnetMask,
	}

	ip := net.ParseIP(decoded.IP.Address)

	// Arbitrarily move 200 addresses into the subnet
	for i := 0; i < 200; i++ {
		ipnetgen.Increment(ip)
		if !subnet.Contains(ip) {
			return "", fmt.Errorf("Could not find unused IP within the subnet %s with subnet mask %s", subnetIp.String(), decoded.IP.Netmask)
		}
	}

	// Check for existing IPs
	ni, err := getNetworkInformation()
	if err != nil {
		return "", err
	}

	// Find the host that IPs are allocated on.  If it
	// does not exist, then make it
	hostData := getHostData(&ni, hostname)
	hostIps := hostData.IPs

	// Iterate over IPs until we find one in the
	// range that is not used.
	ipAddr := ""
	for ; subnet.Contains(ip); ipnetgen.Increment(ip) {
		ipStr := ip.String()
		_, ok := hostIps[ipStr]
		if !ok {
			ipAddr = ipStr
			hostIps[ipStr] = true
			break
		}
	}

	if ipAddr == "" {
		return "", fmt.Errorf("Could not find unused IP within the subnet %s with subnet mask %s", decoded.IP.Address, decoded.IP.Netmask)
	}

	return ipAddr, nil
}

func allocatePort(hostname string) (uint16, error) {
	ni, err := getNetworkInformation()
	if err != nil {
		return 0, err
	}

	hd := getHostData(&ni, hostname)

	// Look for an unallocated port, starting at
	// the default K8s port of 6443.  For now, run
	// all the way up to 65535.  It is likely that
	// this will have to be improved as time goes on.
	var p uint16
	for p = uint16(6443); p < uint16(65535); p = p + 1 {
		_, ok := hd.Ports[p]
		if ok {
			continue
		}
		hd.Ports[p] = true
		break
	}

	if p == 65535 {
		return 0, fmt.Errorf("Could not find unused port")
	}

	return p, nil
}

func addCluster(hostname string, clusterName string, ip string, port uint16) error {
	ni, err := getNetworkInformation()
	if err != nil {
		return err
	}

	_, ok := ni.Clusters[clusterName]
	if ok {
		return fmt.Errorf("Cluster %s already exists on host %s", clusterName, hostname)
	}

	hd := getHostData(&ni, hostname)

	cluster := &ClusterData{
		Host: hostname,
		IP:   ip,
		Port: port,
	}

	// Don't cache localhost values.  These can be
	// reused arbitrarily
	if ip != "127.0.0.1" {
		hd.IPs[ip] = true
	}
	hd.Ports[port] = true
	ni.Clusters[clusterName] = cluster

	return saveNetworkInformation(ni)
}

func removeCluster(clusterName string) error {
	ni, err := getNetworkInformation()
	if err != nil {
		return err
	}

	cluster, ok := ni.Clusters[clusterName]
	if !ok {
		return nil
	}

	hd, ok := ni.Hosts[cluster.Host]
	if !ok {
		return nil
	}

	delete(hd.IPs, cluster.IP)
	delete(hd.Ports, cluster.Port)
	delete(ni.Clusters, clusterName)

	saveNetworkInformation(ni)
	return nil
}
