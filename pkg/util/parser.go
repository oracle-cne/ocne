// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sync"
)

// ScanCloser is a token object that is used to close a scanning routine
type ScanCloser struct {
	Closed bool
	Mutex  sync.Mutex
}

// Close closes a scanner
func (sc *ScanCloser) Close() {
	sc.Mutex.Lock()
	sc.Closed = true
	sc.Mutex.Unlock()
}

// ScanDispatcher is used to process lines from a scanner
type ScanDispatcher interface {
	Dispatch(string)
}


func Scan(reader io.Reader, sd ScanDispatcher) *ScanCloser {
	ret := &ScanCloser{
		Closed: false,
		Mutex: sync.Mutex{},
	}

	go func(rdr io.Reader, sc *ScanCloser) {
		bufReader := bufio.NewReader(rdr)

		for {
			line := ""
			for {
				lineBytes, isPrefix, err := bufReader.ReadLine()
				line = fmt.Sprintf("%s%s", line, string(lineBytes))
				if err != nil {
					break
				}

				if !isPrefix {
					break
				}
			}
			sd.Dispatch(line)

			sc.Mutex.Lock()
			if sc.Closed {
				sc.Mutex.Unlock()
				break
			}
			sc.Mutex.Unlock()
		}
	}(reader, ret)

	return ret
}

// MessageHandler is used to handle multi-line messages
type MessageHandler interface {
	Handle([]string)
}

// MessageDispatcher is a simple object for taking lines from a
// scanner and collecting them into multi-line messages based on
// a regex that defines the start of a new message.
type MessageDispatcher struct {
	StartPattern   *regexp.Regexp
	Handler        MessageHandler
	CurrentMessage []string
}

// NewMessageDispatcher creates a MessageDispatcher
func NewMessageDispatcher(startPattern string, handler MessageHandler) (*MessageDispatcher, error) {
	re, err := regexp.Compile(startPattern)
	if err != nil {
		return nil, err
	}

	return &MessageDispatcher{
		StartPattern: re,
		Handler: handler,
		CurrentMessage: nil,
	}, nil
}

// Dispatch implements the ScanDispatcher interface
func (md *MessageDispatcher) Dispatch(in string) {
	// Check to see if this line starts a new message.  If so, dispatch the
	// current message and start over.
	if md.StartPattern.MatchString(in) && len(md.CurrentMessage) != 0 {
		md.Handler.Handle(md.CurrentMessage)
		md.CurrentMessage = []string{}
	}
	md.CurrentMessage = append(md.CurrentMessage, in)
}

