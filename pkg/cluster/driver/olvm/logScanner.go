// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type OlvmLogHandler struct {
	Errors map[string]bool
}

func NewOlvmLogHandler() *OlvmLogHandler {
	return &OlvmLogHandler{
		Errors: map[string]bool{},
	}
}

var tolerations []*regexp.Regexp = []*regexp.Regexp{
	// '\' is a special character in regex as well as strings, so we need
	// some goop to match '\"'.
	// \\\\\" -> \\ escapes the next backslash \\ literal backslash \" escapes the quote
	// In each pair, the first backslash is for the string escaping.
	regexp.MustCompile("OLVMCluster\\.infrastructure\\.cluster\\.x-k8s\\.io \\\\\".*\\\\\" not found"),
}

func (olh *OlvmLogHandler) Handle(lines []string) {
	// If the message is not an error, ignore it
	if lines[0][0] != 'E' {
		return
	}

	// Error messages are usually split across a few lines.  If it's not,
	// then log an error if the error is not tolerated.
	if len(lines) == 1 {
		for _, r := range tolerations {
			if r.Match([]byte(lines[0])) {
				return
			}
		}
		log.Errorf("Saw unexpected error message in OLVM Cluster API provider logs: %s", lines[0])
		return
	}

	// This message is an error.  Look at it to see if it's an error
	// that has been seen before.  If it has, don't print it again.

	prefix, suffix, _ := strings.Cut(lines[1], " Message: ")
	if prefix == "" {
		log.Errorf("Saw unexpected line in error message in OLVM Cluster API provider logs: %s", lines[1])
		return
	}
	if suffix == "" {
		log.Errorf("Saw unexpected line in error message in OLVM Cluster API provider logs: %s", lines[1])
		return
	}

	// Assemble the message
	lines[1] = fmt.Sprintf("%s %s", prefix, suffix)

	// The key for the message is:
	// - Error message minus request id
	// - Operation name
	// - Endpoint
	opName := ""
	endpoint := ""
	for idx, l := range lines {
		line := strings.TrimSpace(l)
		lines[idx] = line
		if strings.HasPrefix(line, "Operation Name:") {
			opName = line
			continue
		}
		if strings.HasPrefix(line, "Request Endpoint:") {
			endpoint = line
			continue
		}
	}

	if opName == "" {
		log.Errorf("OLVM Cluster API provider log does not have an operation name")
		return
	}
	if endpoint == "" {
		log.Errorf("OLVM Cluster API provider log does not have a request endpoint")
		return
	}

	key := fmt.Sprintf("%s %s %s", lines[1], opName, endpoint)
	_, ok := olh.Errors[key]
	if ok {
		return
	}

	olh.Errors[key] = true

	log.Errorf("Error with OLVM Cluster API provider:\n%s", strings.Join(lines[1:], "\n"))
}
