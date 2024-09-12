// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"github.com/seancfoley/ipaddress-go/ipaddr"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// CreateAndPersistKubernetesCerts creates and persists the cert used by Kubernetes in an OCNE cluster
// The certs are written to the output directory and filenames specified in CertLocation
func CreateAndPersistKubernetesCerts(kubeApiServerIP string, serviceSubnet string, outdir string, options CertOptions) (*CertPairWithPem, error) {
	pair, err := createKubernetesCerts(kubeApiServerIP, serviceSubnet, options)
	if err != nil {
		return nil, err
	}

	loc := CertLocation{
		Directory:              outdir,
		RootCertFilename:       "ca.crt",
		RootPrivateKeyFilename: "ca.key",
		LeafCertFilename:       "admin.crt",
		LeafPrivateKeyFilename: "admin.key",
	}
	err = writeCertPair(pair, loc)
	return pair, err
}

// createKubernetesCerts creates the cert used by OCNE when provisioning a cluster
func createKubernetesCerts(kubeApiServerIP string, serviceSubnet string, options CertOptions) (*CertPairWithPem, error) {
	const (
		local_IP = "127.0.0.1"
	)
	rootConfig := CertConfig{
		CommonName:  kubeApiServerIP,
		CertOptions: options,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(20, 0, 0),
	}
	leafOptions := CertOptions{
		Country: options.Country,
		Org:     "system:masters",
		OrgUnit: options.OrgUnit,
		State:   options.State,
	}

	leafConfig := CertConfig{
		CommonName:  "admin",
		CertOptions: leafOptions,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		DNSNames: []string{
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster",
			"kubernetes.default.svc.cluster.local",
		},
	}
	leafConfig.Org = "system:masters"

	serviceIP, err := getFirstIp(serviceSubnet)
	if err != nil {
		return nil, err
	}
	leafConfig.IPAddresses = append(leafConfig.IPAddresses, serviceIP)

	IPs := []string{kubeApiServerIP, local_IP}
	for _, IP := range IPs {
		leafConfig.IPAddresses = append(leafConfig.IPAddresses, net.ParseIP(IP))
	}

	pemData, err := createCACertAndLeafCert(rootConfig, leafConfig)
	if err != nil {
		log.Errorf("Failed to create Kubernetes Certificate for OCNE: %v", err)
		return nil, err
	}
	return pemData, nil
}

// getFirstIp gets the first IP in a subnet
func getFirstIp(subnet string) (net.IP, error) {
	block := ipaddr.NewIPAddressString(subnet).GetAddress().ToPrefixBlock()
	addr := block.WithoutPrefixLen().GetLower().Increment(1)
	return net.ParseIP(addr.String()), nil
}
