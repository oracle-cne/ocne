// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	"github.com/oracle-cne/ocne/pkg/helm"
	"os"
	"os/exec"
)

// Template takes in a set of options and returns the string that describes the helm template
func Template(opt application.TemplateOptions) ([]byte, error) {
	log.Debug("generating values.yaml")
	var bytesToReturn []byte

	// Get the catalog information along with setting the port-forward
	cat, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: opt.KubeConfigPath,
		CatalogName:    opt.Catalog,
		Pattern:        opt.AppName,
	})
	if err != nil {
		return bytesToReturn, err
	}

	chartReader, err := install.DownloadApplication(cat, opt.AppName, opt.Version)
	if err != nil {
		return nil, err
	}

	// Generate the values for the helm template
	output, err := helm.GetShowValues(chartReader)
	bytesToReturn = output
	return bytesToReturn, err

}

// RunInteractiveMode takes the output of the template function, writes it to a file, and displays it
// It displays it by opening up the text editor specified in the EDITOR environment variable
func RunInteractiveMode(name string, output []byte) error {
	fileName := name + "-values.yaml"
	err := os.WriteFile(name+"-values.yaml", output, 0644)
	if err != nil {
		return err
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR enviroment variable is not set")
	}
	cmd := exec.Command(editor, fileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil

}
