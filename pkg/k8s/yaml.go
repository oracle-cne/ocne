// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"bufio"
	"bytes"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"strings"
)

type ErrorNotExist struct {
}

func (e *ErrorNotExist) Error() string {
	return "does not exist"
}

// Unmarshall a reader containing YAML to a list of unstructured objects
func Unmarshall(reader *bufio.Reader) ([]unstructured.Unstructured, error) {
	buffer := bytes.Buffer{}
	objs := []unstructured.Unstructured{}

	flushBuffer := func() error {
		if buffer.Len() < 1 {
			return nil
		}
		obj := unstructured.Unstructured{Object: map[string]interface{}{}}
		yamlBytes := buffer.Bytes()
		if err := yaml.Unmarshal(yamlBytes, &obj); err != nil {
			return err
		}
		if len(obj.Object) > 0 {
			objs = append(objs, obj)
		}
		buffer.Reset()
		return nil
	}

	eofReached := false
	for {
		// Read the file line by line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// EOF has been reached, but there may be some line data to process
				eofReached = true
			} else {
				return objs, err
			}
		}
		lineStr := string(line)
		// Flush buffer at document break
		// Do not ignore whitespace on the left when searching for sep
		if strings.HasPrefix(lineStr, "---") {
			if err = flushBuffer(); err != nil {
				return objs, err
			}
		} else {
			// Save line to buffer
			if !strings.HasPrefix(lineStr, "#") && len(strings.TrimSpace(lineStr)) > 0 {
				if _, err := buffer.Write(line); err != nil {
					return objs, err
				}
			}
		}
		// if EOF, flush the buffer and return the objs
		if eofReached {
			flushErr := flushBuffer()
			return objs, flushErr
		}
	}
}

// FindIn finds the first object that matches the criteria in a
// multi-doc yaml string.
func FindIn(haystack string, filter func(unstructured.Unstructured) bool) (unstructured.Unstructured, error) {
	candidates, err := Unmarshall(bufio.NewReader(bytes.NewBuffer([]byte(haystack))))
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	for _, c := range candidates {
		if filter(c) {
			return c, nil
		}
	}

	return unstructured.Unstructured{}, &ErrorNotExist{}
}

// FindAll finds all objects that match the criteria in a
// multi-doc yaml string.
func FindAll(haystack string, filter func(unstructured.Unstructured) bool) ([]unstructured.Unstructured, error) {
	candidates, err := Unmarshall(bufio.NewReader(bytes.NewBuffer([]byte(haystack))))
	if err != nil {
		return nil, err
	}

	var ret []unstructured.Unstructured
	for _, c := range candidates {
		if filter(c) {
			ret = append(ret, c)
		}
	}

	return ret, nil
}

// IsNotExist checks if an error represents non-existence
func IsNotExist(err error) bool {
	_, ok := err.(*ErrorNotExist)
	return ok
}
