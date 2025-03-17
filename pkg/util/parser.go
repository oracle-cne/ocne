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

type ScanCloser struct {
	Closed bool
	Mutex  sync.Mutex
}

func (sc *ScanCloser) Close() {
	sc.Mutex.Lock()
	sc.Closed = true
	sc.Mutex.Unlock()
}

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

type MessageHandler interface {
	Handle([]string)
}

type MessageDispatcher struct {
	StartPattern   *regexp.Regexp
	Handler        MessageHandler
	CurrentMessage []string
}

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

func (md *MessageDispatcher) Dispatch(in string) {
	// Check to see if this line starts a new message.  If so, dispatch the
	// current message and start over.
	if md.StartPattern.MatchString(in) && len(md.CurrentMessage) != 0 {
		md.Handler.Handle(md.CurrentMessage)
		md.CurrentMessage = []string{}
	}
	md.CurrentMessage = append(md.CurrentMessage, in)
}

