// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubepki

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/oracle-cne/ocne/pkg/certificate"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/util"
)

type PKIInfo struct {
	CACertPath string
	CAKeyPath  string
	CertsDir   string
}

type KubeconfigRequest struct {
	Path           string
	Host           string
	Port           uint16
	ServiceSubnets []string
}

// GeneratePKI creates the complete set of certificates required
// for a Kubernetes PKI.  This includes the root CA, admin certificates,
// and kubeconfig files.
//
// Each argument creates a kubeconfig with admin privileges.  The first
// in the list is the canonical one for the cluster.  Extra kubeconfigs
// are supplementary and only useful for special cases.
func GeneratePKI(options certificate.CertOptions, canonKr KubeconfigRequest, krs ...KubeconfigRequest) (*PKIInfo, error) {
	bootstrapDirectory, err := file.CreateOcneTempDir("bootstrap_certs")
	if err != nil {
		return nil, err
	}

	certPair, err := certificate.CreateAndPersistKubernetesCerts(canonKr.Host, canonKr.ServiceSubnets, bootstrapDirectory, options)
	if err != nil {
		return nil, err
	}

	// Generate all kubeconfigs
	for _, kr := range append([]KubeconfigRequest{canonKr}, krs...) {
		err = generateKubeconfig(kr.Path, kr.Host, kr.Port, certPair)
		if err != nil {
			return nil, err
		}
	}

	return &PKIInfo{
		CACertPath: filepath.Join(bootstrapDirectory, "ca.crt"),
		CAKeyPath:  filepath.Join(bootstrapDirectory, "ca.key"),
		CertsDir:   bootstrapDirectory,
	}, nil
}

// generateKubeconfig generates a kubeconfig file and writes it to a file
func generateKubeconfig(kubeConfigPath string, kubeAPIServerIP string, port uint16, pair *certificate.CertPairWithPem) error {
	const kubTemplate = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{.RootCACert}}
    server: {{.ServerURL}}
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: {{.ClientCert}}
    client-key-data: {{.ClientKey}}
`
	type templateData struct {
		RootCACert string
		ClientCert string
		ClientKey  string
		ServerURL  string
	}

	t, err := template.New("template").Parse(kubTemplate)
	if err != nil {
		return err
	}

	encoder := base64.StdEncoding
	var buf bytes.Buffer
	var tData = templateData{
		RootCACert: encoder.EncodeToString(pair.RootCertResult.CertPEM),
		ClientCert: encoder.EncodeToString(pair.LeafCertResult.CertPEM),
		ClientKey:  encoder.EncodeToString(pair.LeafCertResult.PrivateKeyPEM),
		ServerURL:  fmt.Sprintf("https://%s:%d", util.GetURIAddress(kubeAPIServerIP), port),
	}

	err = t.Execute(&buf, &tData)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(kubeConfigPath), 0700)
	if err != nil {
		return err
	}
	return os.WriteFile(kubeConfigPath, buf.Bytes(), 0644)
}
