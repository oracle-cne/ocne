// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capture

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture/gvr"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Indentation when the resource is marshalled as Json
	JSONIndent = "  "

	// The prefix used for the json.MarshalIndent
	JSONPrefix = ""
)

// captureByNamespace collects the Kubernetes resources from the specified namespace, as JSON files
func captureByNamespace(cs *captureSync, dynamicClient dynamic.Interface, outDir, namespace string, redact bool) {
	goCaptureDynamicRes(cs, gvr.K8sNamespacedResources, dynamicClient, outDir, namespace, redact)
	goCaptureDynamicRes(cs, gvr.CapiNamespacedResources, dynamicClient, outDir, namespace, redact)
	goCaptureDynamicRes(cs, gvr.CertmanagerNamespacedResources, dynamicClient, outDir, namespace, redact)
	goCaptureDynamicRes(cs, gvr.IstioNamespacedResources, dynamicClient, outDir, namespace, redact)
	goCaptureDynamicRes(cs, gvr.PromNamespacedResources, dynamicClient, outDir, namespace, redact)

}

// captureClusterWide collects the Kubernetes resources that are cluster wide
func captureClusterWide(cs *captureSync, dynamicClient dynamic.Interface, outDir string, redact bool) {
	goCaptureDynamicRes(cs, gvr.K8sClusterResources, dynamicClient, outDir, "", redact)
	goCaptureDynamicRes(cs, gvr.CapiClusterResources, dynamicClient, outDir, "", redact)
	goCaptureDynamicRes(cs, gvr.CertmanagerClusterResources, dynamicClient, outDir, "", redact)
	goCaptureDynamicRes(cs, gvr.IstioClusterResources, dynamicClient, outDir, "", redact)
	goCaptureDynamicRes(cs, gvr.PromClusterResources, dynamicClient, outDir, "", redact)

}

// goCaptureDynamicRes will create a goroutine for every gvr to dump the gvr resource list manifests to a file
func goCaptureDynamicRes(cs *captureSync, gvrs []schema.GroupVersionResource, dynamicClient dynamic.Interface, outDir string, namespace string, redact bool) {
	for i, _ := range gvrs {
		cs.wg.Add(1)

		// block until a slot is free in the channel buffer.
		cs.channel <- 0
		go func() {
			err := captureDynamicRes(dynamicClient, gvrs[i], outDir, namespace, redact)
			if err != nil {
				log.Errorf(err.Error())
			}
			<-cs.channel
			cs.wg.Done()
		}()
	}
}

// captureDynamicRes will get the list of all resources that match the gvr and write the JSON file
func captureDynamicRes(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, outDir string, namespace string, redact bool) error {
	var list *unstructured.UnstructuredList
	var err error
	if len(namespace) == 0 {
		list, err = dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	} else {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	}
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		log.Errorf(fmt.Sprintf("An error occurred while listing %s in namespace %s: %s\n", gvr.Resource, namespace, err.Error()))
		return nil
	}
	if len(list.Items) > 0 {
		var fname string
		if gvr.Group == "" {
			fname = fmt.Sprintf("%s.json", strings.ToLower(gvr.Resource))
		} else {
			fname = fmt.Sprintf("%s.%s.json", strings.ToLower(gvr.Resource), strings.ToLower(gvr.Group))
		}

		if err = writeJSONToFile(list, namespace, fname, outDir, redact); err != nil {
			return err
		}
	}
	return nil
}

// writeJSONToFile write the JSON data to a file
func writeJSONToFile(v interface{}, namespace, resourceFile, outDir string, redact bool) error {
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err := os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("Error creating the directory %s: %s", outDir, err.Error())
		}
	}

	var res = filepath.Join(outDir, resourceFile)
	f, err := os.OpenFile(res, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf(createFileError, res, err.Error())
	}
	defer f.Close()

	resJSON, _ := json.MarshalIndent(v, JSONPrefix, JSONIndent)
	if redact {
		_, err = f.WriteString(SanitizeString(string(resJSON), nil))
	} else {
		_, err = f.WriteString(string(resJSON))
	}

	if err != nil {
		return fmt.Errorf("Error writing the file %s: %s", res, err.Error())
	}
	return nil
}