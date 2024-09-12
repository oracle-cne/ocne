// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capture

import (
	"bufio"
	"bytes"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/info"
	"os"
	"path/filepath"
)

const clusterInfoFile = "cluster-info.out"

func CaptureClusterInfo(kubeClient kubernetes.Interface, outDir string, skipRedact bool) {
	b := bytes.Buffer{}
	writer := bufio.NewWriter(&b)

	// get the cluster info
	err := info.Info(info.Options{
		KubeClient:  kubeClient,
		RootDumpDir: outDir,
		Writer:      writer,
	})
	if err != nil {
		log.Errorf(err.Error())
	}
	writer.Flush()

	// create the output file
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err := os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			log.Errorf("Error creating the directory %s: %s", outDir, err.Error())
		}
	}

	var res = filepath.Join(outDir, clusterInfoFile)
	fOut, err := os.OpenFile(res, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf(createFileError, res, err.Error())
	}
	defer fOut.Close()

	if !skipRedact {
		sanitizeLines(bytes.NewReader(b.Bytes()), bufio.NewWriter(fOut))
	} else {
		writeWithoutSanitize(bytes.NewReader(b.Bytes()), bufio.NewWriter(fOut))
	}
}
