// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"io"
	"net/http"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

// GetNamespace returns the object storage namespace for this tenancy
func GetNamespace(profile string) (string, error) {
	ctx := context.Background()
	c, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(common.CustomProfileConfigProvider("", profile))
	if err != nil {
		return "", err
	}

	resp, err := c.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
	if err != nil {
		return "", err
	}

	return *resp.Value, nil
}

// UploadObject uploads the contents of a stream to an object storage bucket
func UploadObject(bucketName string, objectName string, profile string, contentLen int64, content io.Reader, metadata map[string]string) error {
	namespace, err := GetNamespace(profile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	c, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(common.CustomProfileConfigProvider("", profile))
	if err != nil {
		return err
	}

	// Apparently this disables timeouts for big uploads?
	// See here: https://github.com/oracle/oci-go-sdk/blob/master/example/example_objectstorage_test.go#L66
	c.HTTPClient = &http.Client{}

	request := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketName,
		ObjectName:    &objectName,
		ContentLength: &contentLen,
		PutObjectBody: io.NopCloser(content),
		OpcMeta:       metadata,
	}
	_, err = c.PutObject(ctx, request)
	return err
}
