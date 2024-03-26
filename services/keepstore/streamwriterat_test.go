// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"sync"

	. "gopkg.in/check.v1"
)

var _ = Suite(&streamWriterAtSuite{})

type streamWriterAtSuite struct{}

func (s *streamWriterAtSuite) TestPartSizes(c *C) {
	for partsize := 1; partsize < 5; partsize++ {
		for writesize := 1; writesize < 5; writesize++ {
			for datasize := 1; datasize < 100; datasize += 13 {
				for bufextra := 0; bufextra < 5; bufextra++ {
					c.Logf("=== partsize %d writesize %d datasize %d bufextra %d", partsize, writesize, datasize, bufextra)
					outbuf := bytes.NewBuffer(nil)
					indata := make([]byte, datasize)
					for i := range indata {
						indata[i] = byte(i)
					}
					swa := newStreamWriterAt(outbuf, partsize, make([]byte, datasize+bufextra))
					var wg sync.WaitGroup
					for pos := 0; pos < datasize; pos += writesize {
						pos := pos
						wg.Add(1)
						go func() {
							defer wg.Done()
							endpos := pos + writesize
							if endpos > datasize {
								endpos = datasize
							}
							swa.WriteAt(indata[pos:endpos], int64(pos))
						}()
					}
					wg.Wait()
					swa.Close()
					c.Check(outbuf.Bytes(), DeepEquals, indata)
				}
			}
		}
	}
}

func (s *streamWriterAtSuite) TestOverflow(c *C) {
	for offset := -1; offset < 2; offset++ {
		buf := make([]byte, 50)
		swa := newStreamWriterAt(bytes.NewBuffer(nil), 20, buf)
		_, err := swa.WriteAt([]byte("foo"), int64(len(buf)+offset))
		c.Check(err, NotNil)
		err = swa.Close()
		c.Check(err, IsNil)
	}
}

func (s *streamWriterAtSuite) TestIncompleteWrite(c *C) {
	for _, partsize := range []int{20, 25} {
		for _, bufsize := range []int{50, 55, 60} {
			for offset := 0; offset < 3; offset++ {
				swa := newStreamWriterAt(bytes.NewBuffer(nil), partsize, make([]byte, bufsize))
				_, err := swa.WriteAt(make([]byte, 1), 49)
				c.Check(err, IsNil)
				_, err = swa.WriteAt(make([]byte, 46), int64(offset))
				c.Check(err, IsNil)
				err = swa.Close()
				c.Check(err, NotNil)
				c.Check(swa.WroteAt(), Equals, 47)
				if offset == 0 {
					c.Check(swa.Wrote(), Equals, 40/partsize*partsize)
				} else {
					c.Check(swa.Wrote(), Equals, 0)
				}
			}
		}
	}
}
