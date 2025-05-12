// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"regexp"
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

var tolerations = []*regexp.Regexp{
	// '\' is a special character in regex as well as strings, so we need
	// some goop to match '\"'.
	// \\\\\" -> \\ escapes the next backslash \\ literal backslash \" escapes the quote
	// In each pair, the first backslash is for the string escaping.
	regexp.MustCompile("OLVMCluster\\.infrastructure\\.cluster\\.x-k8s\\.io \\\\\".*\\\\\" not found"),
}

func (olh *LogHandler) Handle(lines []string) {
	// If the message is not an error, ignore it
	if !strings.Contains(lines[0], "ERROR") {
		return
	}

	// Check if this error message is tolerated
	for _, r := range tolerations {
		if r.Match([]byte(lines[0])) {
			return
		}
	}

	// This message is an error.  Look at it to see if it's an error
	// that has been seen before.  If it has, don't print it again.

	// The format of the text messages is text fields separated by tabs.
	// Assuming the format to be:
	//   part[0] - timestamp
	//   part[1] - log level
	//   part[2] - caller
	//   part[3] - error text
	//   part[4] - additional info (e.g., name, namespace, GVK)
	parts := strings.Split(lines[0], "\t")

	// The key for the message is:
	// - Error message
	key := fmt.Sprintf("%s", strings.TrimSpace(parts[3]))
	_, ok := olh.Errors[key]
	if ok {
		return
	}

	olh.Errors[key] = true

	log.Errorf("Error with OLVM Cluster API provider:\n%s", lines[0])
}
