// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"errors"
	"fmt"
	v1Apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	crtpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetResource(restConf *rest.Config, u *unstructured.Unstructured) error {
	client, err := crtpkg.New(restConf, crtpkg.Options{})
	if err != nil {
		return err
	}

	err = client.Get(context.TODO(), crtpkg.ObjectKey{
		Namespace: u.GetNamespace(),
		Name:      u.GetName(),
	}, u)
	return nil
}

func GetResourceByIdentifier(restConf *rest.Config, group string, version string, kind string, name string, namespace string) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version,
	})
	u.SetName(name)
	u.SetNamespace(namespace)
	err := GetResource(restConf, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func DeleteResourceByIdentifier(restConf *rest.Config, group string, version string, kind string, name string, namespace string) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version,
	})
	u.SetName(name)
	u.SetNamespace(namespace)
	err := GetResource(restConf, u)
	if err != nil {
		return err
	}
	err = DeleteResource(restConf, u)
	return err
}

func DeleteResource(restConf *rest.Config, u *unstructured.Unstructured) error {
	client, err := crtpkg.New(restConf, crtpkg.Options{})
	if err != nil {
		return err
	}
	return client.Delete(context.TODO(), u)
}

func GrabDeployment(objectsToSearch []unstructured.Unstructured) (v1Apps.Deployment, int, error) {
	for idx, object := range objectsToSearch {
		if strings.ToLower(object.GetKind()) == "deployment" {
			var deployment v1Apps.Deployment
			converter := runtime2.DefaultUnstructuredConverter
			err := converter.FromUnstructured(object.UnstructuredContent(), &deployment)
			if err != nil {
				return v1Apps.Deployment{}, 0, err
			}
			return deployment, idx, nil
		}
	}
	tmp := fmt.Sprintf("deployment not found")
	return v1Apps.Deployment{}, 0, errors.New(tmp)
}

func GrabContainer(objectsToSearch []v1.Container, name string) (v1.Container, int, error) {
	for idx, container := range objectsToSearch {
		if container.Name == name {
			return container, idx, nil
		}
	}
	tmp := fmt.Sprintf("container %s not found", name)
	return v1.Container{}, 0, errors.New(tmp)
}