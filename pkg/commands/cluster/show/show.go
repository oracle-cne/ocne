// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package show

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oracle-cne/ocne/pkg/cluster/cache"
)

func yamlTag(f reflect.StructField) string {
	yml, ok := f.Tag.Lookup("yaml")
	if !ok {
		return f.Name
	}

	yml = strings.Split(yml, ",")[0]
	if yml == "" {
		return f.Name
	}
	return yml
}

func getChild(parent interface{}, path []string) (interface{}, error) {
	v := reflect.ValueOf(parent)
	t := reflect.TypeOf(parent)

	if len(path) == 0 {
		return parent, nil
	}

	childName := path[0]
	path = path[1:]
	var child interface{}
	switch t.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			// This map came from yaml, so the map key
			// is definitely a string.
			if strings.EqualFold(iter.Key().String(), childName) {
				child = iter.Value().Interface()
				break
			}
		}
	case reflect.Slice:
		idx, err := strconv.Atoi(childName)
		if err != nil {
			return nil, err
		}
		parentSlice := parent.([]interface{})
		if len(parentSlice) < idx {
			return nil, fmt.Errorf("List has less than %d members", idx)
		}
		child = parentSlice[idx]
	case reflect.Struct:
		fields := reflect.VisibleFields(t)
		for _, f := range fields {
			tag := yamlTag(f)
			if strings.EqualFold(tag, childName) {
				child = v.FieldByName(f.Name).Interface()
				break
			}
		}
	default:
		return nil, fmt.Errorf("Cannot get the child of a %s", t.Kind())
	}

	if child == nil {
		return nil, fmt.Errorf("Could not find element %s", childName)
	}

	return getChild(child, path)
}

func Show(name string, all bool, field string) error {
	clusterCache, err := cache.GetCache()
	if err != nil {
		return err
	}

	clusterConfig := clusterCache.Get(name)
	if clusterConfig == nil {
		return fmt.Errorf("Cluster %s does not exist", name)
	}

	if all {
		allBytes, err := yaml.Marshal(clusterConfig)
		if err != nil {
			return err
		}

		fmt.Println(string(allBytes))
		return nil
	}

	child, err := getChild(*clusterConfig, strings.Split(field, "."))
	if err != nil {
		return err
	}

	// If this field is not marshallable, just print it
	switch reflect.TypeOf(child).Kind() {
	case reflect.Map, reflect.Slice, reflect.Struct:
		break
	default:
		fmt.Println(child)
		return nil
	}

	fieldBytes, err := yaml.Marshal(child)
	if err != nil {
		return err
	}

	fmt.Println(string(fieldBytes))

	return nil
}
