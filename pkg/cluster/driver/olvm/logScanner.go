// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type LogHandler struct {
	Errors map[string]bool
}

func NewLogHandler() *LogHandler {
	return &LogHandler{
		Errors: map[string]bool{},
	}
}

func (olh *LogHandler) Handle(lines []string) {
	// If the message is not an error, ignore it
	if !strings.Contains(lines[0], "ERROR") {
		return
	}

	// The format of the unstructured log messages is text fields separated by tabs.
	// Assuming the format to be:
	//   part[0] - timestamp
	//   part[1] - log level
	//   part[2] - caller
	//   part[3] - error text
	//   part[4] - additional info (e.g., name, namespace, GVK)
	parts := strings.Split(lines[0], "\t")

	// The key for the message is:
	// - Error message
	// Keep track of errors we have seen before and only log them once.
	key := fmt.Sprintf("%s", strings.TrimSpace(parts[3]))
	_, ok := olh.Errors[key]
	if ok {
		return
	}

	olh.Errors[key] = true

	// The complete error may span more than one line.
	// Print all the lines as a single message to the console.
	log.Errorf("Error with OLVM Cluster API provider:\n%s", strings.Join(lines[0:], "\n"))
}
