// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"
	"strings"
)

// HelmValueFileConstructor creates a YAML file from a set of key value pairs
func HelmValueFileConstructor(kvs []KeyValue) (string, error) {
	yamlObject := map[string]interface{}{}
	for _, kv := range kvs {
		// replace unwanted characters in the value to avoid splitting
		ignoreChars := ",[.{}"
		for _, char := range ignoreChars {
			kv.Value = strings.Replace(kv.Value, string(char), "\\"+string(char), -1)
		}

		composedStr := fmt.Sprintf("%s=%s", kv.Key, kv.Value)
		var err error
		if kv.SetString {
			err = strvals.ParseIntoString(composedStr, yamlObject)
		} else {
			err = strvals.ParseInto(composedStr, yamlObject)
		}
		if err != nil {
			return "", err
		}
	}

	yamlFile, err := yaml.Marshal(yamlObject)
	if err != nil {
		return "", err
	}
	return string(yamlFile), nil
}
