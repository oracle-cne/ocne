// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubepug

import (
	"embed"
	"fmt"

	"github.com/kubepug/kubepug/pkg/kubepug"
	k8sinput "github.com/kubepug/kubepug/pkg/kubepug/input/k8s"
	"github.com/kubepug/kubepug/pkg/results"
	"github.com/kubepug/kubepug/pkg/store/generatedstore"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

//go:embed data.json
var data embed.FS

func GetDatabase() ([]byte, error) {

	return data.ReadFile("data.json")
}

func GetDeprecations(restConfig *rest.Config, k8sVersion string) (*results.Result, error) {
	db, err := GetDatabase()
	if err != nil {
		return nil, err
	}

	gs, err := generatedstore.NewGeneratedStoreFromBytes(db, generatedstore.StoreConfig{
		MinVersion: k8sVersion,
	})
	if err != nil {
		return nil, err
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	input := &k8sinput.K8sInput{
		Store: gs,
		Client: dynClient,
		DiscoveryClient: discClient,
		IncludePrefixGroup: []string{".k8s.io"},
		IgnoreExactGroup: []string{},
	}

	// kubepug uses logrus to log and prints stuff that we don't
	// need to see
	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.FatalLevel)
	ret, err := kubepug.GetDeprecations(input)
	logrus.SetLevel(origLevel)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func FormatItem(item *results.Item) string {
	return fmt.Sprintf("    %s/%s", item.Namespace, item.ObjectName)
}

func FormatItems(res *results.Result) ([]string, []string) {
	deprec := []string{"The APIs for the following resources are being removed"}
	removed := []string{"The APIs for the following resources are being removed"}

	for _, d := range res.DeprecatedAPIs {
		if len(d.Items) == 0 {
			continue
		}

		deprec = append(deprec, "  %s/%s/%s", d.Group, d.Version, d.Kind)
		for _, i := range d.Items {
			deprec = append(deprec, FormatItem(&i))
		}
	}
	for _, r := range res.DeletedAPIs {
		if len(r.Items) == 0 {
			continue
		}

		removed = append(removed, "  %s/%s/%s", r.Group, r.Version, r.Kind)
		for _, i := range r.Items {
			removed = append(removed, FormatItem(&i))
		}
	}

	// If there weren't actually any deprecations or removals, don't report them
	if len(deprec) == 1 {
		deprec = nil
	}
	if len(removed) == 1 {
		removed = nil
	}

	return deprec, removed
}
