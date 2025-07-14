// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"

	apiex "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiexv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apival "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Given a CRD and a resource, validate the resource.  The first return
// value indicates if the resource is valid.  The second is a list of
// errors.  The third is a list of warnings
func ValidateCustomResource(crd *apiex.CustomResourceDefinition, res *unstructured.Unstructured) (bool, []error, []error) {
	gvk := res.GroupVersionKind()

	// Check the GVK
	if crd.Spec.Group != gvk.Group {
		return false, []error{fmt.Errorf("CRD group %s does not match resource group %s for %s", crd.Spec.Group, gvk.Group, gvk.String())}, nil
	}
	if crd.Spec.Names.Kind != gvk.Kind {
		return false, []error{fmt.Errorf("CRD kind %s does not match resource kind %s for %s", crd.Spec.Names.Kind, gvk.Kind, gvk.String())}, nil
	}

	// Now find the right version.  If no correct version exists, then
	// it is considered invalid.
	version, err := apiex.GetSchemaForVersion(crd, gvk.Version)
	if version == nil {
		return false, []error{fmt.Errorf("CRD does not have version %s for %s", gvk.Version, gvk.String())}, nil
	}

	// At this point it is known that the resource can be validated against
	// the CRD.  So do that.
	validator, _, err := apival.NewSchemaValidator(version.OpenAPIV3Schema)
	if err != nil {
		return false, []error{err}, nil
	}

	ret := validator.Validate(res)
	if len(ret.Errors) != 0 {
		return false, ret.Errors, ret.Warnings
	}

	return true, nil, ret.Warnings
}

func CRDFromBytes(in []byte) (*apiex.CustomResourceDefinition, error) {
	crdv1 := &apiexv1.CustomResourceDefinition{}
	ret := &apiex.CustomResourceDefinition{}

	apiexv1.SetDefaults_CustomResourceDefinition(crdv1)
	err := yaml.Unmarshal(in, crdv1)
	if err != nil {
		return nil, err
	}

	err = apiexv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crdv1, ret, nil)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
