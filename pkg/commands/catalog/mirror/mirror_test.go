// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mirror

import (
	"fmt"
	"os"
	"testing"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type imageCatalog map[string][]byte

func (c imageCatalog) GetCharts(string) (*catalog.Catalog, error) {
	return nil, fmt.Errorf("unexpected index request")
}

func (c imageCatalog) GetChart(name, version string) ([]byte, error) {
	data, ok := c[name+version]
	if !ok {
		return nil, fmt.Errorf("missing chart")
	}
	return data, nil
}

func TestFilterByImage(t *testing.T) {
	const repository = "container-registry.oracle.com/os/oraclelinux"
	digest := repository + "@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	connection := imageCatalog{}
	for version, image := range map[string]string{"1.0.0": repository + ":8", "2.0.0": repository + ":9", "3.0.0": digest, "4.0.0": repository} {
		archive, err := chartutil.Save(&chart.Chart{
			Metadata:  &chart.Metadata{APIVersion: "v2", Name: "sample", Version: version},
			Templates: []*chart.File{{Name: "templates/pod.yaml", Data: []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: sample\nspec:\n  containers:\n  - name: main\n    image: " + image + "\n")}},
		}, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		connection["sample"+version], err = os.ReadFile(archive)
		if err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name, image string
		count       int
		wantError   bool
	}{
		{"repository", repository, 4, false},
		{"tag", repository + ":8", 1, false},
		{"digest", digest, 1, false},
		{"implicit latest", repository + ":latest", 1, false},
		{"unknown tag", repository + ":10", 0, false},
		{"different repository", repository + "-other", 0, false},
		{"invalid", "bad image", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := &catalog.Catalog{Connection: connection, ChartEntries: map[string][]catalog.ChartMeta{
				"sample": {{Name: "sample", Version: "1.0.0"}, {Name: "sample", Version: "2.0.0"}, {Name: "sample", Version: "3.0.0"}, {Name: "sample", Version: "4.0.0"}},
			}}
			err := FilterByImage(found, tc.image, "")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if !tc.wantError && len(found.ChartEntries["sample"]) != tc.count {
				t.Fatalf("matched %d versions, want %d", len(found.ChartEntries["sample"]), tc.count)
			}
		})
	}
	t.Run("chart fetch failure", func(t *testing.T) {
		found := &catalog.Catalog{Connection: connection, ChartEntries: map[string][]catalog.ChartMeta{"missing": {{Name: "missing", Version: "1.0.0"}}}}
		if err := FilterByImage(found, repository, ""); err != nil {
			t.Fatal(err)
		}
		if len(found.ChartEntries) != 0 {
			t.Fatal("unavailable chart should be skipped")
		}
	})
}
