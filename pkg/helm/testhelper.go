// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"io"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/registry"
	"helm.sh/helm/v4/pkg/release"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
)

type CreateReleaseFnType func(name string, releaseStatus release.Status) *release.Release

func CreateActionConfig(includeRelease bool, releaseName string, releaseStatus release.Status, createReleaseFn CreateReleaseFnType) (*action.Configuration, error) {

	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	cfg := &action.Configuration{
		Releases:       storage.Init(driver.NewMemory()),
		KubeClient:     &fake.FailingKubeClient{PrintingKubeClient: fake.PrintingKubeClient{Out: io.Discard}},
		Capabilities:   common.DefaultCapabilities,
		RegistryClient: registryClient,
		Log:            debugLog,
	}
	if includeRelease {
		testRelease := createReleaseFn(releaseName, releaseStatus)
		err = cfg.Releases.Create(testRelease)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
