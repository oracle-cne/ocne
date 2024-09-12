// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"time"
)

type retryFunc func(interface{}) (interface{}, bool, error)

// ExponentialRetryImpl executes a functor at some interval until is either
// succeeds, fails in a non-recoverable way, or a timeout is reached. The
// given functor is called with the given arguments each interval.  'start'
// is the first duration to wait.  'max' is the maximum duration.  'factor'
// is the amount to increase the wait each iteration.  'timeout' is the
// last time to start a request.  Due to the fact that a functor call may
// be long, this function may take longer than the given timeout.
func ExponentialRetryImpl(ftor retryFunc, arg interface{}, start time.Duration, max time.Duration, factor time.Duration, timeout time.Duration) (interface{}, bool, error) {
	begin := time.Now()

	var err error
	var failFast bool
	var ret interface{}
	wait := start
	incr := factor
	for time.Since(begin) < timeout {
		ret, failFast, err = ftor(arg)
		if failFast && err != nil {
			return ret, failFast, err
		}

		if err == nil {
			return ret, false, nil
		}

		time.Sleep(wait)

		if wait < max {
			wait = start + factor
			incr = incr * factor
		} else {
			wait = max
		}
	}

	return ret, false, err
}

// ExponentialRetry executes a functor at some interval until is either
// succeeds, fails in a non-recoverable way, or a timeout is reached. The
// given functor is called with the given arguments each interval.
func ExponentialRetry(ftor retryFunc, arg interface{}) (interface{}, bool, error) {
	return ExponentialRetryTimeout(ftor, arg, 10*time.Second)
}

// ExponentialRetryTimeout executes a functor at some interval until is either
// succeeds, fails in a non-recoverable way, or a timeout is reached. The
// given functor is called with the given arguments each interval.
func ExponentialRetryTimeout(ftor retryFunc, arg interface{}, timeout time.Duration) (interface{}, bool, error) {
	return ExponentialRetryImpl(ftor, arg, 10*time.Millisecond, time.Second, time.Second, timeout)
}

// LinearRetryImpl executes a functor every 'wait' until it either succeeds, fails in
// a way that should not be retries, or until the timeout is reached.  If the functor
// succeeds, the function returns false with no error.  If the functor fails in a way
// that should not be retried, the function returns true with an error.  If the function
// times out, it returns false as well as the last error from the functor.
func LinearRetryImpl(ftor retryFunc, arg interface{}, wait time.Duration, timeout time.Duration) (interface{}, bool, error) {
	return ExponentialRetryImpl(ftor, arg, wait, wait, 0, timeout)
}

// LinearRetry executes a functor every 'wait' until it either succeeds, fails in
// a way that should not be retries, or until the timeout of 10 seconds is reached.  If the functor
// succeeds, the function returns false with no error.  If the functor fails in a way
// that should not be retried, the function returns true with an error.  If the function
// times out, it returns false as well as the last error from the functor.
func LinearRetry(ftor retryFunc, arg interface{}) (interface{}, bool, error) {
	return LinearRetryTimeout(ftor, arg, 10*time.Second)
}

// LinearRetryTimeout executes a functor every 'wait' until it either succeeds, fails in
// a way that should not be retries, or until the timeout is reached.  If the functor
// succeeds, the function returns false with no error.  If the functor fails in a way
// that should not be retried, the function returns true with an error.  If the function
// times out, it returns false as well as the last error from the functor.
func LinearRetryTimeout(ftor retryFunc, arg interface{}, timeout time.Duration) (interface{}, bool, error) {
	return LinearRetryImpl(ftor, arg, 100*time.Millisecond, timeout)
}
