// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logger

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"runtime"
)

func ErrorWithWrapper(format string, a ...any) {
	initialMessageToWrite := fmt.Sprintf(format, a...)
	initialMessageToWrite = initialMessageToWrite + determineStackTrace()
	log.Error(initialMessageToWrite)

}

// This function determines a stack trace
func determineStackTrace() string {
	stackTraceString := ""
	// The index of frames is set to 2 as this allows us to skip the helper function calls in the logger package
	indexOfFrame := 2
	for {
		_, file, line, ok := runtime.Caller(indexOfFrame)
		if !ok {
			break
		}
		stackTraceString = stackTraceString + fmt.Sprintf("\n%s:%d", file, line)
		indexOfFrame = indexOfFrame + 1
	}
	return stackTraceString
}
