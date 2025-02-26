// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"io"
	"net/http"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	log "github.com/sirupsen/logrus"
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

// EnsureObject ensures that an object exists in object storage and that the
// object is the same as the one that is desired.  If the object exists and
// is the same, then the function simply returns.  If not, it uploads the
// object to the given bucket and object name.
func EnsureObject(bucketName string, objectName string, profile string, contentLen int64, content io.ReadCloser, metadata map[string]string) error {
	namespace, err := GetNamespace(profile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	c, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(common.CustomProfileConfigProvider("", profile))
	if err != nil {
		return err
	}

	// Check to see if an object already exists.
	resp, err := c.ListObjects(ctx, objectstorage.ListObjectsRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketName,
		Prefix:        &objectName,
	})
	if err != nil {
		return err
	}

	// Check if the objects are the same.  A rough estimate is fine for now.
	for _, o := range resp.ListObjects.Objects {
		if *o.Name == objectName {
			log.Debugf("Object already exists")
			return nil
		}
	}

	return UploadObject(bucketName, objectName, profile, contentLen, content, metadata)
}

// UploadObject uploads the contents of a stream to an object storage bucket
func UploadObject(bucketName string, objectName string, profile string, contentLen int64, content io.ReadCloser, metadata map[string]string) error {
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
		PutObjectBody: content,
		OpcMeta:       metadata,
	}
	_, err = c.PutObject(ctx, request)
	return err

}
