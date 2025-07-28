// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logutils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/oracle-cne/ocne/pkg/util"
)

func logScanner(s *bufio.Scanner, l func(string), filter func(string, interface{}) bool, arg interface{}) {
	for s.Scan() {
		txt := s.Text()
		if filter == nil || filter(txt, arg) {
			l(txt)
		}
	}
	err := s.Err()
	if err != nil && err != io.ErrClosedPipe {
		log.Errorf("Error reading from stream: %v", err)
	}
}

type Stream struct {
	Reader io.Reader
	Filter func(string, interface{}) bool
	Arg    interface{}
}

// Reads two streams, typically the output of a function call
// as we  as its error stream, and prints them.  The info/debug
// stream is logged at the debug level.  The error stream is logged
// at the error level.  If a nil value is provided for a particular
// reader, no goroutines are spawned and no attempt is made to
// read the Reader.
//
// This method is asynchronous.  It creates two goroutines to perform
// the logging.  The caller is responsible for closing the writers that
// correspond to the two readers.  Failure to do so prevents the goroutines
// that write the log messages from terminating.
func LogDebugAndErrors(stdout *Stream, stderr *Stream) {
	// Start two readers, one for stdout and one for stderr
	if !util.IsNil(stdout.Reader) {
		stdoutScanner := bufio.NewScanner(stdout.Reader)
		go logScanner(stdoutScanner, Debug, stdout.Filter, stdout.Arg)
	}
	if !util.IsNil(stderr.Reader) {
		stderrScanner := bufio.NewScanner(stderr.Reader)
		go logScanner(stderrScanner, Error, stderr.Filter, stderr.Arg)
	}
}

// Waiter defines a function to wait for and a message
// to display while waiting.
type Waiter struct {
	WaitFunction    func(interface{}) error
	Args            interface{}
	Message         string
	MessageFunction func(interface{}) string
	Error           error
	done            bool
	mutex           sync.RWMutex
}

// Info is a wrapper around log.Info()
func Info(s string) {
	log.Info(s)
}

// Debug is a wrapper around log.Debug()
func Debug(s string) {
	log.Debug(s)
}

// Error is a wrapper around log.Error
func Error(s string) {
	log.Error(s)
}

func getMsg(waiter *Waiter) string {
	if waiter.MessageFunction != nil {
		return waiter.MessageFunction(waiter.Args)
	}
	return waiter.Message
}

func waitWithStatus(waiter *Waiter) {
	err := waiter.WaitFunction(waiter.Args)

	waiter.mutex.Lock()
	waiter.done = true
	waiter.Error = err
	log.Debugf("Wait done")
	waiter.mutex.Unlock()
}

// shouldBackup determines if WaitFor
// should back up lines each loop
func shouldBackup() (bool, error) {
	return util.FileIsTTY(os.Stdout)
}

// backup moves the cursor up n lines
//
// ^[[&dA is the VT-100 escape code to move the
// cursor up %d lines.  In GO, ^[ is \x1b
func backup(n int) {
	fmt.Printf("\x1b[%dA", n)
}

var colorReset = "\x1b[0m"
var colorYellow = "\x1b[33m"
var colorGreen = "\x1b[32m"

var regular = "\x1b[0m"

var clearLine = "\x1b[K"

// printDone prints a message for completed jobs
// formatted based on if it was successful or not.
func printDone(logFn func(string), w *Waiter) {
	if w.Error != nil {
		log.Errorf("%s: %s%s", getMsg(w), w.Error, clearLine)
	} else {
		// ^[[2K is the VT-100 escape code to clear all text
		// to the right of the cursor.
		logFn(fmt.Sprintf("%s: %s%s%s%s", getMsg(w), colorGreen, "ok", colorReset, clearLine))
	}
}

// WaitForWithLevel starts some goroutines and pretty-prints a
// message for each. The level is need to prevent backup unless messages,
// are displayed. Returns true if an Error occurred
// for any of the waiters.
func WaitForWithLevel(logFn func(string), level log.Level, waiters []*Waiter) bool {
	return waitFor(logFn, level, waiters)
}

// WaitFor starts some goroutines and pretty-prints a
// message for each.  Returns true if an Error occurred
// for any of the waiters.
func WaitFor(logFn func(string), waiters []*Waiter) bool {
	return waitFor(logFn, log.InfoLevel, waiters)
}

var waitStrings []string = []string{
	colorYellow + "waiting",
	colorYellow + "waiting.",
	colorYellow + "waiting..",
	colorYellow + "waiting...",
	colorYellow + "waiting ..",
	colorYellow + "waiting  .",
}

func waitString(msg string, iter int) string {
	idx := iter % len(waitStrings)
	return fmt.Sprintf("%s: %s%s%s", msg, waitStrings[idx], colorReset, clearLine)
}

// waitFor starts some goroutines and pretty-prints a
// message for each.  Returns true if an Error occurred
// for any of the waiters.
func waitFor(logFn func(string), level log.Level, waiters []*Waiter) bool {
	haveError := false
	doBackup, err := shouldBackup()
	if log.GetLevel() < level {
		// only backup if messages are being logged
		doBackup = false
	}

	if err != nil {
		log.Error(err)
		return true
	}

	// Kick off our waiters
	for _, w := range waiters {
		go waitWithStatus(w)
	}

	// Wait for everything, logging as they go
	loops := 0
	for len(waiters) > 0 {
		// Back up the terminal if applicable
		done := []*Waiter{}
		notDone := []*Waiter{}
		for _, w := range waiters {
			w.mutex.RLock()

			if w.done {
				// Log completion message
				done = append(done, w)

			} else {
				notDone = append(notDone, w)
			}

			if w.Error != nil {
				haveError = true
			}

			w.mutex.RUnlock()
		}
		for _, w := range done {
			printDone(logFn, w)
		}
		for _, w := range notDone {
			logFn(waitString(getMsg(w), loops))
		}
		loops = loops + 1

		waiters = notDone
		if len(waiters) == 0 {
			break
		} else if doBackup {
			backup(len(waiters))
		}
		time.Sleep(500 * time.Millisecond)

	}

	return haveError
}

// WaitForSerial runs a series of functors in serial, waiting for
// each to complete to before starting the next one.  It pretty-prints
// a log message for each, even if the functor has not started yet.
// Returns true if an Error has occurred for any of the waiters.
func WaitForSerial(logFn func(string), waiters []*Waiter) bool {
	doBackup, err := shouldBackup()
	if err != nil {
		log.Error(err)
		return true
	}

	haveError := false
	for len(waiters) > 0 {
		for _, w := range waiters {
			logFn(waitString(getMsg(w), 0))
		}

		if doBackup {
			backup(len(waiters))
		}

		w := waiters[0]

		go waitWithStatus(w)
		loops := 0
		for {
			w.mutex.RLock()
			if w.done {
				w.mutex.RUnlock()
				break
			}
			w.mutex.RUnlock()
			logFn(waitString(getMsg(w), loops))
			loops = loops + 1
			if doBackup {
				backup(1)
			}
			time.Sleep(500 * time.Millisecond)
		}

		printDone(logFn, w)

		if w.Error != nil {
			haveError = true
		}

		waiters = waiters[1:]
	}

	return haveError
}

// ProgressBar takes a percentage value and
// returns a string that indicates that percentage
// has completed in a nice way.
func ProgressBar(complete float32) string {
	barLen := 10
	barStep := 100 / barLen
	completeInt := int(complete)
	ret := "["
	for i := 0; i < barLen; i++ {
		if (i * barStep) < completeInt {
			ret = ret + "#"
		} else {
			ret = ret + " "
		}
	}
	ret = ret + "]"
	return ret
}
