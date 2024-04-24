// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"io"
	"math/rand"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&writeBufferSuite{})

type writeBufferSuite struct {
}

// 1000 / 96.3 ns/op = 10.384 GB/s
func (s *writeBufferSuite) Benchmark_1KBWrites(c *C) {
	wb := newWriteBuffer(io.Discard, 1<<20)
	in := make([]byte, 1000)
	for i := 0; i < c.N; i++ {
		wb.Write(in)
	}
	wb.Close()
}

func (s *writeBufferSuite) TestRandomizedSpeedsAndSizes(c *C) {
	for i := 0; i < 20; i++ {
		insize := rand.Intn(1 << 26)
		bufsize := rand.Intn(1 << 26)
		if i < 2 {
			// make sure to test edge cases
			bufsize = i
		} else if insize/bufsize > 1000 {
			// don't waste too much time testing tiny
			// buffer / huge content
			insize = bufsize*1000 + 123
		}
		c.Logf("%s: insize %d bufsize %d", c.TestName(), insize, bufsize)

		in := make([]byte, insize)
		b := byte(0)
		for i := range in {
			in[i] = b
			b++
		}

		out := &bytes.Buffer{}
		done := make(chan struct{})
		pr, pw := io.Pipe()
		go func() {
			n, err := slowCopy(out, pr, rand.Intn(8192)+1)
			c.Check(err, IsNil)
			c.Check(n, Equals, int64(insize))
			close(done)
		}()
		wb := newWriteBuffer(pw, bufsize)
		n, err := slowCopy(wb, bytes.NewBuffer(in), rand.Intn(8192)+1)
		c.Check(err, IsNil)
		c.Check(n, Equals, int64(insize))
		c.Check(wb.Close(), IsNil)
		c.Check(pw.Close(), IsNil)
		<-done
		c.Check(out.Len(), Equals, insize)
		for i := 0; i < out.Len() && i < len(in); i++ {
			if out.Bytes()[i] != in[i] {
				c.Errorf("content mismatch at byte %d", i)
				break
			}
		}
	}
}

func slowCopy(dst io.Writer, src io.Reader, bufsize int) (int64, error) {
	wrote := int64(0)
	buf := make([]byte, bufsize)
	for {
		time.Sleep(time.Duration(rand.Intn(100) + 1))
		n, err := src.Read(buf)
		if n > 0 {
			n, err := dst.Write(buf[:n])
			wrote += int64(n)
			if err != nil {
				return wrote, err
			}
		}
		if err == io.EOF {
			return wrote, nil
		}
		if err != nil {
			return wrote, err
		}
	}
}
