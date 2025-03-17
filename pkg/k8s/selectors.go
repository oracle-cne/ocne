// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package k8s

import (
	"k8s.io/apimachinery/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LabelsToSelector creates a selector string from a map of labels.
func LabelsToSelector(labelMap map[string]string) string {
	ls := &metav1.LabelSelector{
		MatchLabels: labelMap,
	}

	lsm, _ := metav1.LabelSelectorAsMap(ls)

	return labels.SelectorFromSet(lsm).String()
}

func stringMapSubset(set map[string]string, subset map[string]string) bool {
	for k, v := range subset {
		setVal, ok := set[k]
		if !ok {
			return false
		}

		if setVal != v {
			return false
		}
	}

	return true
}
