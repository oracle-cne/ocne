// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capture

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// discoverGVRs discovers all the GVRs on the system
func discoverGVRs(kubeClient kubernetes.Interface, includeConfigMaps bool) (clusterGVRs []schema.GroupVersionResource, namespacedGVRS []schema.GroupVersionResource, err error) {
	// Get namespaced GVRs
	resList, err := kubeClient.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return nil, nil, err
	}
	namespacedGVRS = buildGRVs(resList, nil, includeConfigMaps)

	// Build the set of namespaced GVRs so that they are skipped for cluster resources
	namespacedSet := make(map[string]bool)
	for _, gvr := range namespacedGVRS {
		namespacedSet[gvr.String()] = true
	}
	resList, err = kubeClient.Discovery().ServerPreferredResources()
	if err != nil {
		return nil, nil, err
	}
	clusterGVRs = buildGRVs(resList, namespacedSet, includeConfigMaps)

	return
}

// buildGRVs builds the GroupVersionResource from the APIResourceList
func buildGRVs(resList []*metav1.APIResourceList, excludeResources map[string]bool, includeConfigMaps bool) []schema.GroupVersionResource {
	// Exclude the following resources.  The secret and configmap are intentionally excluded.
	// v1 ComponentStatus is deprecated
	// The other resources cannot be fetched, the server returns the error:
	//   "the server does not allow this method on the requested resource"
	//
	excludeResNames := map[string]bool{
		"bindings":                  true,
		"configmaps":                true,
		"componentstatuses":         true,
		"localsubjectaccessreviews": true,
		"multiclustercomponents":    true,
		"multiclusterconfigmaps":    true,
		"multiclustersecrets":       true,
		"secrets":                   true,
		"selfsubjectaccessreviews":  true,
		"selfsubjectrulesreviews":   true,
		"selfsubjectreviews":        true,
		"subjectaccessreviews":      true,
		"tokenreviews":              true,
	}

	if includeConfigMaps {
		delete(excludeResNames, "configmaps")
	}

	gvrs := []schema.GroupVersionResource{}
	for _, r := range resList {
		var group string
		var version string
		segs := strings.Split(r.GroupVersion, "/")
		if len(segs) == 1 {
			// legacy v1 resources like pod have the version only
			group = ""
			version = segs[0]
		} else {
			group = segs[0]
			version = segs[1]
		}

		for _, a := range r.APIResources {
			if _, ok := excludeResNames[a.Name]; ok {
				continue
			}
			gvr := schema.GroupVersionResource{
				Group:    group,
				Version:  version,
				Resource: a.Name,
			}
			// only append if this is excluded
			if _, ok := excludeResources[gvr.String()]; !ok {
				gvrs = append(gvrs, gvr)
			}
		}
	}

	return gvrs
}
