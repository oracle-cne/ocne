// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ls

import (
	"helm.sh/helm/v3/pkg/release"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

// List takes in a set of options and returns a list of releases from a given namespace or from all namespace
func List(opt application.LsOptions) ([]*release.Release, error) {
	var releases []*release.Release

	kubeInfo, err := client.CreateKubeInfo(opt.KubeConfigPath)
	if err != nil {
		return releases, err
	}
	if opt.All {
		releases, err = helm.GetReleasesAllNamespaces(kubeInfo)
		return releases, err
	} else {
		if opt.Namespace == "" {
			opt.Namespace, err = client.GetNamespaceFromConfig(opt.KubeConfigPath)
			if err != nil {
				return releases, err
			}
		}

		releases, err = helm.GetReleases(kubeInfo, opt.Namespace)
		return releases, err
	}
}
