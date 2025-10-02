// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package embedded

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
)

var charts embed.FS

func FindCandidates(prefix string) ([]string, error) {
	fents, err := charts.ReadDir("charts")
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, fent := range fents {
		log.Debugf("Have candidate %s", fent.Name())
		if strings.HasPrefix(fent.Name(), prefix) {
			log.Debugf("Candidate %s matched %s", fent.Name(), prefix)
			ret = append(ret, fmt.Sprintf("charts/%s", fent.Name()))
		}
	}

	log.Debugf("Candidates for %s are: %+v", prefix, ret)
	return ret, nil
}

func readerFromPath(path string) (io.Reader, error) {
	chartBytes, err := charts.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(chartBytes), nil
}

// GetChartAtVersion finds a chart tarball by name and version.  The returned
// io.Reader reads the tarball.
func GetChartAtVersion(name string, version string) (io.Reader, error) {
	cands, err := FindCandidates(fmt.Sprintf("%s-%s", name, version))
	if err != nil {
		return nil, err
	}

	if len(cands) == 0 {
		return nil, fmt.Errorf("No charts named %s at version %s could be found", name, version)
	} else if len(cands) > 1 {
		return nil, fmt.Errorf("Multiple charts named %s have version %s", name, version)
	}

	return readerFromPath(cands[0])
}

// GetIndex returns the index.yaml for this repo
func GetIndex() (io.Reader, error) {
	return readerFromPath("charts/index.yaml")
}
