// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"

	. "gopkg.in/check.v1"
)

type HashcheckSuiteSuite struct{}

// Gocheck boilerplate
var _ = Suite(&HashcheckSuiteSuite{})

func (h *HashcheckSuiteSuite) TestRead(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	{
		r, w := io.Pipe()
		hcr := HashCheckingReader{r, md5.New(), hash}
		go func() {
			w.Write([]byte("foo"))
			w.Close()
		}()
		p, err := ioutil.ReadAll(hcr)
		c.Check(len(p), Equals, 3)
		c.Check(err, Equals, nil)
	}

	{
		r, w := io.Pipe()
		hcr := HashCheckingReader{r, md5.New(), hash}
		go func() {
			w.Write([]byte("bar"))
			w.Close()
		}()
		p, err := ioutil.ReadAll(hcr)
		c.Check(len(p), Equals, 3)
		c.Check(err, Equals, BadChecksum)
	}
}

func (h *HashcheckSuiteSuite) TestWriteTo(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	{
		bb := bytes.NewBufferString("foo")
		hcr := HashCheckingReader{bb, md5.New(), hash}
		r, w := io.Pipe()
		done := make(chan bool)
		go func() {
			p, err := ioutil.ReadAll(r)
			c.Check(len(p), Equals, 3)
			c.Check(err, Equals, nil)
			done <- true
		}()

		n, err := hcr.WriteTo(w)
		w.Close()
		c.Check(n, Equals, int64(3))
		c.Check(err, Equals, nil)
		<-done
	}

	{
		bb := bytes.NewBufferString("bar")
		hcr := HashCheckingReader{bb, md5.New(), hash}
		r, w := io.Pipe()
		done := make(chan bool)
		go func() {
			p, err := ioutil.ReadAll(r)
			c.Check(len(p), Equals, 3)
			c.Check(err, Equals, nil)
			done <- true
		}()

		n, err := hcr.WriteTo(w)
		w.Close()
		c.Check(n, Equals, int64(3))
		c.Check(err, Equals, BadChecksum)
		<-done
	}

	// If WriteTo stops early due to a write error, return the
	// write error (not "bad checksum").
	{
		input := bytes.NewBuffer(make([]byte, 1<<26))
		hcr := HashCheckingReader{input, md5.New(), hash}
		r, w := io.Pipe()
		r.Close()
		n, err := hcr.WriteTo(w)
		c.Check(n, Equals, int64(0))
		c.Check(err, NotNil)
		c.Check(err, Not(Equals), BadChecksum)
	}
}
