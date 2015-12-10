package main

import (
	"bytes"
	"testing/iotest"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollisionSuite{})

type CollisionSuite struct{}

func (s *CollisionSuite) TestCollisionOrCorrupt(c *check.C) {
	fooMD5 := "acbd18db4cc2f85cedef654fccc4a4d8"

	c.Check(collisionOrCorrupt(fooMD5, []byte{'f'}, []byte{'o'}, bytes.NewBufferString("o")),
		check.Equals, CollisionError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f'}, nil, bytes.NewBufferString("oo")),
		check.Equals, CollisionError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f'}, []byte{'o', 'o'}, nil),
		check.Equals, CollisionError)
	c.Check(collisionOrCorrupt(fooMD5, nil, []byte{}, bytes.NewBufferString("foo")),
		check.Equals, CollisionError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'o', 'o'}, nil, bytes.NewBufferString("")),
		check.Equals, CollisionError)
	c.Check(collisionOrCorrupt(fooMD5, nil, nil, iotest.NewReadLogger("foo: ", iotest.DataErrReader(iotest.OneByteReader(bytes.NewBufferString("foo"))))),
		check.Equals, CollisionError)

	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'o', 'o'}, nil, bytes.NewBufferString("bar")),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'o'}, nil, nil),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{}, nil, bytes.NewBufferString("")),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'O'}, nil, bytes.NewBufferString("o")),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'O', 'o'}, nil, nil),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'o'}, []byte{'O'}, nil),
		check.Equals, DiskHashError)
	c.Check(collisionOrCorrupt(fooMD5, []byte{'f', 'o'}, nil, bytes.NewBufferString("O")),
		check.Equals, DiskHashError)

	c.Check(collisionOrCorrupt(fooMD5, []byte{}, nil, iotest.TimeoutReader(iotest.OneByteReader(bytes.NewBufferString("foo")))),
		check.Equals, iotest.ErrTimeout)
}
