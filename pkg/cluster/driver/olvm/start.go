// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/application"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	oci2 "github.com/oracle-cne/ocne/pkg/cluster/template/oci"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/olvmutil"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	capiclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

const (
	credsUsernameKey = "username"
	credsPasswordKey = "password"
	credsScopeKey    = "scope"
	caCrtBaseKey     = "ca.crt"
)

// Start creates an OLVM CAPI cluster which includes a set of control plane nodes and worker nodes.
func (cad *OlvmDriver) Start() (bool, bool, error) {
	// If there is a need to generate a template, do so.
	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return false, false, err
		}

		cad.ClusterResources = cdi
	}

	if err := oci2.ValidateClusterResources(cad.ClusterResources); err != nil {
		return false, false, err
	}

	// A fair bit of metadata is anchored by the Cluster
	// object in the bundle of Cluster API resources.  Fetch it
	// to a) make sure it exists and b) fetch useful information
	clusterObj, err := cad.getClusterObject()
	if err != nil {
		return false, false, err
	}

	cad.ResourceNamespace = clusterObj.GetNamespace()

	// Apply the given yaml.
	restConfig, clientIface, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return false, false, err
	}

	err = k8s.CreateNamespaceIfNotExists(clientIface, cad.ResourceNamespace)
	if err != nil {
		return false, false, err
	}

	// create the ovirt secret, configmap, etc.
	err = cad.createRequiredResources(clientIface)
	if err != nil {
		return false, false, err
	}

	// Get logs for the OLVM controller
	pods, err := application.PodsForApplication(constants.OLVMCAPIRelease, constants.OLVMCAPIOperatorNamespace, cad.BootstrapKubeConfig)
	log.Debugf("Have %d OLVM CAPI pods", len(pods))
	if err != nil {
		return false, false, err
	}
	podLogs := []*util.ScanCloser{}
	for _, op := range pods {
		podLog, err := k8s.GetPodLogs(clientIface, op, "")
		if err != nil {
			return false, false, err
		}
		re := "^[A-Z][0-9]+"
		md, err := util.NewMessageDispatcher(re, NewLogHandler())
		if err != nil {
			err = fmt.Errorf("Internal error: regex \"%s\" does not compile: %v", re, err)
			return false, false, err
		}
		podLogs = append(podLogs, util.Scan(podLog, md))
	}
	defer func(toClose []*util.ScanCloser) {
		for _, tc := range toClose {
			tc.Close()
		}
	}(podLogs)

	// create all the CAPI resources
	log.Info("Applying Cluster API resources")
	err = cad.applyResources(restConfig)
	if err != nil {
		return false, false, err
	}

	// Get the kubeconfig.  This is done by finding a secret
	// that has the same label as the top level Cluster resource.
	clusterName, _ := clusterObj.GetLabels()[ClusterNameLabel]

	var kubeconfig string
	logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for kubeconfig",
			WaitFunction: func(i interface{}) error {
				kubeconfig, err = cad.waitForKubeconfig(clientIface, clusterName)
				if err != nil {
					return err
				}
				return nil
			},
		},
	})

	if err != nil {
		return false, false, err
	}

	err = os.WriteFile(cad.KubeConfig, []byte(kubeconfig), 0700)
	if err != nil {
		return false, false, err
	}

	_, kubeClient, err := client.GetKubeClient(cad.KubeConfig)

	// Wait for the cluster to start
	_, err = k8s.WaitUntilGetNodesSucceeds(kubeClient)
	if err != nil {
		return false, false, err
	}

	// Once the cluster has started, install all the necessary applications
	// for the workload cluster.
	workloadApps, err := cad.getWorkloadClusterApplications(restConfig, kubeClient)
	if err != nil {
		return false, false, err
	}

	log.Info("Installing applications into workload cluster")
	err = install.InstallApplications(workloadApps, cad.KubeConfig, cad.Config.Quiet)
	if err != nil {
		return false, false, err
	}

	return false, false, nil
}

// PostStart installs the CAPI controllers and dependent apps in a self-managed cluster.
func (cad *OlvmDriver) PostStart() error {
	// If the cluster is not self-managed, then the configuration is
	// complete.
	if !cad.ClusterConfig.Providers.Olvm.SelfManaged {
		return nil
	}

	_, kubeClient, err := client.GetKubeClient(cad.KubeConfig)
	if err != nil {
		return err
	}

	// Install the Cluster API controllers into the new cluster
	capiApplications, err := cad.getApplications()
	if err != nil {
		return err
	}

	err = install.InstallApplications(capiApplications, cad.KubeConfig, cad.Config.Quiet)
	if err != nil {
		return err
	}

	// Wait for controllers to settle.  Nodes should have their cloud
	// provider taints removed.  There should also be at least one
	// node that has no control plane taints.
	err = cad.waitForControllers(kubeClient)
	if err != nil {
		return err
	}

	// Move the resources to the new cluster
	capiClient, err := capiclient.New(context.TODO(), "")
	if err != nil {
		return nil
	}

	log.Info("Migrating Cluster API resources into self-managed cluster")
	err = capiClient.Move(context.TODO(), capiclient.MoveOptions{
		FromKubeconfig: capiclient.Kubeconfig{Path: cad.BootstrapKubeConfig, Context: ""},
		ToKubeconfig:   capiclient.Kubeconfig{Path: cad.KubeConfig, Context: ""},
		Namespace:      cad.ResourceNamespace,
		DryRun:         false,
	})
	if err != nil {
		return err
	}

	// Scale the bootstrap cluster controllers back up.
	return nil
}

// waitForKubeconfig waits for the kubeconfig secret to be created.
func (cad *OlvmDriver) waitForKubeconfig(client kubernetes.Interface, clusterName string) (string, error) {
	var kubeconfig string
	kcfgSecretIface, _, err := util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		log.Debugf("Looking for secrets in %s with label %s = %s", cad.ResourceNamespace, ClusterNameLabel, clusterName)
		secrets, err := k8s.FindSecretsByLabelKeyVal(client, cad.ResourceNamespace, ClusterNameLabel, clusterName)
		if err != nil {
			log.Debugf("Error finding secret: %+v", err)
			return nil, false, err
		}
		if len(secrets.Items) == 0 {
			log.Debugf("No secrets found")
			return nil, false, fmt.Errorf("no kubeconfig found for cluster %s in namespace %s", clusterName, cad.ResourceNamespace)
		}

		// Find the secret that looks kubeconfig'y
		for _, s := range secrets.Items {
			if strings.Contains(s.ObjectMeta.Name, "kubeconfig") {
				return &s, false, nil
			}
		}
		return nil, false, fmt.Errorf("no kubeconfig found for cluster %s in namespace %s", clusterName, cad.ResourceNamespace)
	}, nil, 20*time.Minute)

	if err != nil {
		return "", err
	}

	kcfgSecret, ok := kcfgSecretIface.(*v1.Secret)
	if !ok {
		return "", fmt.Errorf("internal error: kubeconfig secret is not a secret")
	}

	// Get the kubeconfig string out of the secret
	kcfgBytes, ok := kcfgSecret.Data["value"]
	if !ok {
		return "", fmt.Errorf("%s is not a valid kubeconfig secret", kcfgSecret.ObjectMeta.Name)
	}
	kubeconfig = string(kcfgBytes)

	return kubeconfig, nil
}

// createRequiredResources creates the required resources needed for an OLVM CAPI cluster.
func (cad *OlvmDriver) createRequiredResources(kubeClient kubernetes.Interface) error {
	// Get the creds
	credmap, err := getCreds()
	if err != nil {
		return err
	}

	secretNsn := cad.credSecretNsn()
	k8s.DeleteSecret(kubeClient, secretNsn.Namespace, secretNsn.Name)
	err = k8s.CreateSecret(kubeClient, secretNsn.Namespace, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNsn.Name,
			Namespace: secretNsn.Namespace,
		},
		Data: credmap,
		Type: "Opaque",
	})
	if err != nil {
		return err
	}

	// get the CA
	if !cad.ClusterConfig.Providers.Olvm.OlvmAPIServer.InsecureSkipTLSVerify {
		caMap, err := GetCAMap(&cad.ClusterConfig.Providers.Olvm)
		if err != nil {
			return err
		}

		cmNsn := cad.caConfigMapNsn()
		k8s.DeleteConfigmap(kubeClient, cmNsn.Namespace, cmNsn.Name)
		if len(caMap) > 0 {
			err = k8s.CreateConfigmap(kubeClient, &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmNsn.Name,
					Namespace: cmNsn.Namespace,
				},
				Data: caMap,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCAMap gets a map of oVirt CA strings from the config, either inline or from a file.
func GetCAMap(prov *types.OlvmProvider) (map[string]string, error) {
	caMap := map[string]string{}
	if prov.OlvmAPIServer.ServerCA != "" && prov.OlvmAPIServer.ServerCAPath != "" {
		return caMap, fmt.Errorf("The OLVM Provider cannot specify both ovirtApiCA and ovirtApiCAPath")
	}
	if prov.OlvmAPIServer.ServerCA != "" {
		caMap[caCrtBaseKey] = prov.OlvmAPIServer.ServerCA
		return caMap, nil
	}

	if prov.OlvmAPIServer.ServerCAPath != "" {
		caFiles := strings.Split(prov.OlvmAPIServer.ServerCAPath, ",")
		baseKey := caCrtBaseKey
		for i, caFile := range caFiles {
			f, err := file.AbsDir(strings.TrimSpace(caFile))
			if err != nil {
				return caMap, err
			}
			by, err := os.ReadFile(f)
			if err != nil {
				return caMap, fmt.Errorf("Error reading OLVM Provider oVirt CA file: %v", err)
			}
			caMap[baseKey] = string(by)

			// Update the key for the next time through the loop
			baseKey = fmt.Sprintf("%s%d", caCrtBaseKey, i+1)
		}
		return caMap, nil
	}
	log.Info("The OLVM Provider was not configured with ovirtApiCA or ovirtApiCAPath, assuming the CA is already in the trust store")
	return caMap, nil
}

// getCreds gets the oVirt creds from a set of ENV vars.
func getCreds() (map[string][]byte, error) {
	username := os.Getenv(EnvUsername)
	if username == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM username", EnvUsername)
	}
	password := os.Getenv(EnvPassword)
	if password == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM password", EnvPassword)
	}
	scope := os.Getenv(EnvScope)
	if scope == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM username", EnvScope)
	}

	return map[string][]byte{
		credsUsernameKey: []byte(username),
		credsPasswordKey: []byte(password),
		credsScopeKey:    []byte(scope),
	}, nil

}

func (cad *OlvmDriver) credSecretNsn() *types.NamespacedName {
	return olvmutil.CredSecretNsn(cad.ClusterConfig)
}

func (cad *OlvmDriver) caConfigMapNsn() *types.NamespacedName {
	return olvmutil.CaConfigMapNsn(cad.ClusterConfig)
}
