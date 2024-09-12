// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package util

import (
	"bytes"
	"text/template"
)

func yesno(in bool) string {
	if in {
		return "yes"
	}
	return "no"
}

func TemplateToString(templateString string, contents interface{}) (string, error) {
	return TemplateToStringWithFuncs(templateString, contents, nil)
}

func TemplateToStringWithFuncs(templateString string, contents interface{}, funcs map[string]any) (string, error) {
	if funcs == nil {
		funcs = map[string]any{}
	}
	funcs["yesno"] = yesno
	tmpl := template.New("template").Funcs(funcs)
	tmpl, err := tmpl.Parse(templateString)
	if err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	err = tmpl.Execute(&buf, contents)
	if err != nil {
		return "", err
	}

	return buf.String(), err
}
