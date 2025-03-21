// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// GetCRDs gets all CRDs defined in the cluster
func GetCRDs(restConf *rest.Config) (*apiextv1.CustomResourceDefinitionList, error) {
	client, err := cs.NewForConfig(restConf)
	if err != nil {
		return nil, err
	}

	return client.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
}

// GetCRDsWithAnnotations get a list of CRDs that have the given set
// of annotations.
func GetCRDsWithAnnotations(restConf *rest.Config, annots map[string]string) ([]*apiextv1.CustomResourceDefinition, error) {
	crds, err := GetCRDs(restConf)
	if err != nil {
		return nil, err
	}

	ret := []*apiextv1.CustomResourceDefinition{}
	for _, crd := range crds.Items {
		if stringMapSubset(crd.Annotations, annots) {
			ret = append(ret, &crd)
		}
	}
	return ret, nil
}

// DeleteCRD deletes a CRD
func DeleteCRD(restConf *rest.Config, cr *apiextv1.CustomResourceDefinition) error {
	client, err := cs.NewForConfig(restConf)
	if err != nil {
		return err
	}
	return client.ApiextensionsV1().CustomResourceDefinitions().Delete(context.TODO(), cr.Name, metav1.DeleteOptions{})
}
