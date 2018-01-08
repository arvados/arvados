// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package cmdtest provides tools for testing command line tools.
package cmdtest

import (
	"io"
	"io/ioutil"
	"os"

	check "gopkg.in/check.v1"
)

// LeakCheck tests for output being leaked to os.Stdout and os.Stderr
// that should be sent elsewhere (e.g., the stdout and stderr streams
// passed to a cmd.RunFunc).
//
// It redirects os.Stdout and os.Stderr to a tempfile, and returns a
// func, which the caller is expected to defer, that restores os.* and
// checks that the tempfile is empty.
//
// Example:
//
//	func (s *Suite) TestSomething(c *check.C) {
//		defer cmdtest.LeakCheck(c)()
//		// ... do things that shouldn't print to os.Stderr or os.Stdout
//	}
func LeakCheck(c *check.C) func() {
	tmpfiles := map[string]*os.File{"stdout": nil, "stderr": nil}
	for i := range tmpfiles {
		var err error
		tmpfiles[i], err = ioutil.TempFile("", "")
		c.Assert(err, check.IsNil)
		err = os.Remove(tmpfiles[i].Name())
		c.Assert(err, check.IsNil)
	}

	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = tmpfiles["stdout"], tmpfiles["stderr"]
	return func() {
		os.Stdout, os.Stderr = stdout, stderr

		for i, tmpfile := range tmpfiles {
			c.Log("checking %s", i)
			_, err := tmpfile.Seek(0, io.SeekStart)
			c.Assert(err, check.IsNil)
			leaked, err := ioutil.ReadAll(tmpfile)
			c.Assert(err, check.IsNil)
			c.Check(string(leaked), check.Equals, "")
		}
	}
}
