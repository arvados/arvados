// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package asyncbuf

import (
	"crypto/md5"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&Suite{})

type Suite struct{}

func (s *Suite) TestNoWrites(c *check.C) {
	b := NewBuffer(nil)
	r1 := b.NewReader()
	r2 := b.NewReader()
	b.Close()
	s.checkReader(c, r1, []byte{}, nil, nil)
	s.checkReader(c, r2, []byte{}, nil, nil)
}

func (s *Suite) TestNoReaders(c *check.C) {
	b := NewBuffer(nil)
	n, err := b.Write([]byte("foobar"))
	err2 := b.Close()
	c.Check(n, check.Equals, 6)
	c.Check(err, check.IsNil)
	c.Check(err2, check.IsNil)
}

func (s *Suite) TestWriteReadClose(c *check.C) {
	done := make(chan bool, 2)
	b := NewBuffer(nil)
	n, err := b.Write([]byte("foobar"))
	c.Check(n, check.Equals, 6)
	c.Check(err, check.IsNil)
	r1 := b.NewReader()
	r2 := b.NewReader()
	go s.checkReader(c, r1, []byte("foobar"), nil, done)
	go s.checkReader(c, r2, []byte("foobar"), nil, done)
	time.Sleep(time.Millisecond)
	c.Check(len(done), check.Equals, 0)
	b.Close()
	<-done
	<-done
}

func (s *Suite) TestPrefillWriteCloseRead(c *check.C) {
	done := make(chan bool, 2)
	b := NewBuffer([]byte("baz"))
	n, err := b.Write([]byte("waz"))
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)
	b.Close()
	r1 := b.NewReader()
	go s.checkReader(c, r1, []byte("bazwaz"), nil, done)
	r2 := b.NewReader()
	go s.checkReader(c, r2, []byte("bazwaz"), nil, done)
	<-done
	<-done
}

func (s *Suite) TestWriteReadCloseRead(c *check.C) {
	done := make(chan bool, 1)
	b := NewBuffer(nil)
	r1 := b.NewReader()
	go s.checkReader(c, r1, []byte("bazwazqux"), nil, done)

	b.Write([]byte("bazwaz"))

	r2 := b.NewReader()
	r2.Read(make([]byte, 3))

	b.Write([]byte("qux"))
	b.Close()

	s.checkReader(c, r2, []byte("wazqux"), nil, nil)
	<-done
}

func (s *Suite) TestReadAtEOF(c *check.C) {
	buf := make([]byte, 8)

	b := NewBuffer([]byte{1, 2, 3})

	r := b.NewReader()
	n, err := r.Read(buf)
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)

	// Reading zero bytes at EOF, but before Close(), doesn't
	// block or error
	done := make(chan bool)
	go func() {
		defer close(done)
		n, err = r.Read(buf[:0])
		c.Check(n, check.Equals, 0)
		c.Check(err, check.IsNil)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Error("timeout")
	}

	b.Close()

	// Reading zero bytes after Close() returns EOF
	n, err = r.Read(buf[:0])
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)

	// Reading from start after Close() returns 3 bytes, then EOF
	r = b.NewReader()
	n, err = r.Read(buf)
	c.Check(n, check.Equals, 3)
	if err != nil {
		c.Check(err, check.Equals, io.EOF)
	}
	n, err = r.Read(buf[:0])
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)
	n, err = r.Read(buf)
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)
}

func (s *Suite) TestCloseWithError(c *check.C) {
	errFake := errors.New("it's not even a real error")

	done := make(chan bool, 1)
	b := NewBuffer(nil)
	r1 := b.NewReader()
	go s.checkReader(c, r1, []byte("bazwazqux"), errFake, done)

	b.Write([]byte("bazwaz"))

	r2 := b.NewReader()
	r2.Read(make([]byte, 3))

	b.Write([]byte("qux"))
	b.CloseWithError(errFake)

	s.checkReader(c, r2, []byte("wazqux"), errFake, nil)
	<-done
}

// Write n*n bytes, n at a time; read them into n goroutines using
// varying buffer sizes; compare checksums.
func (s *Suite) TestManyReaders(c *check.C) {
	const n = 256

	b := NewBuffer(nil)

	expectSum := make(chan []byte)
	go func() {
		hash := md5.New()
		buf := make([]byte, n)
		for i := 0; i < n; i++ {
			time.Sleep(10 * time.Nanosecond)
			rand.Read(buf)
			b.Write(buf)
			hash.Write(buf)
		}
		expectSum <- hash.Sum(nil)
		b.Close()
	}()

	gotSum := make(chan []byte)
	for i := 0; i < n; i++ {
		go func(bufSize int) {
			got := md5.New()
			io.CopyBuffer(got, b.NewReader(), make([]byte, bufSize))
			gotSum <- got.Sum(nil)
		}(i + n/2)
	}

	expect := <-expectSum
	for i := 0; i < n; i++ {
		c.Check(expect, check.DeepEquals, <-gotSum)
	}
}

func (s *Suite) BenchmarkOneReader(c *check.C) {
	s.benchmarkReaders(c, 1)
}

func (s *Suite) BenchmarkManyReaders(c *check.C) {
	s.benchmarkReaders(c, 100)
}

func (s *Suite) benchmarkReaders(c *check.C, readers int) {
	var n int64
	t0 := time.Now()

	buf := make([]byte, 10000)
	rand.Read(buf)
	for i := 0; i < 10; i++ {
		b := NewBuffer(nil)
		go func() {
			for i := 0; i < c.N; i++ {
				b.Write(buf)
			}
			b.Close()
		}()

		var wg sync.WaitGroup
		for i := 0; i < readers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				nn, _ := io.Copy(ioutil.Discard, b.NewReader())
				atomic.AddInt64(&n, int64(nn))
			}()
		}
		wg.Wait()
	}
	c.Logf("%d bytes, %.0f MB/s", n, float64(n)/time.Since(t0).Seconds()/1000000)
}

func (s *Suite) checkReader(c *check.C, r io.Reader, expectData []byte, expectError error, done chan bool) {
	buf, err := ioutil.ReadAll(r)
	c.Check(err, check.Equals, expectError)
	c.Check(buf, check.DeepEquals, expectData)
	if done != nil {
		done <- true
	}
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
