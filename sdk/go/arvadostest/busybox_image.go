// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package arvadostest

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"

	. "gopkg.in/check.v1"
)

// BusyboxDockerImage downloads the busybox:uclibc docker image
// (busybox_uclibc.tar) from cache.arvados.org into a temporary file
// and returns the temporary file name.
func BusyboxDockerImage(c *C) string {
	fnm := "busybox_uclibc.tar"
	cachedir := c.MkDir()
	cachefile := cachedir + "/" + fnm
	if _, err := os.Stat(cachefile); err == nil {
		return cachefile
	}

	f, err := ioutil.TempFile(cachedir, "")
	c.Assert(err, IsNil)
	defer f.Close()
	defer os.Remove(f.Name())

	resp, err := http.Get("https://cache.arvados.org/" + fnm)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)
	err = os.Rename(f.Name(), cachefile)
	c.Assert(err, IsNil)

	return cachefile
}
