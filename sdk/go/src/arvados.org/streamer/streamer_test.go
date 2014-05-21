package streamer

import (
	. "gopkg.in/check.v1"
	"io"
	"testing"
	"time"
)

// Gocheck boilerplate
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&StandaloneSuite{})

// Standalone tests
type StandaloneSuite struct{}

func (s *StandaloneSuite) TestReadIntoBuffer(c *C) {
	ReadIntoBufferHelper(c, 225)
	ReadIntoBufferHelper(c, 224)
}

func HelperWrite128andCheck(c *C, buffer []byte, writer io.Writer, slices chan nextSlice) {
	out := make([]byte, 128)
	for i := 0; i < 128; i += 1 {
		out[i] = byte(i)
	}
	writer.Write(out)
	s1 := <-slices
	c.Check(len(s1.slice), Equals, 128)
	c.Check(s1.reader_error, Equals, nil)
	for i := 0; i < 128; i += 1 {
		c.Check(s1.slice[i], Equals, byte(i))
	}
	for i := 0; i < len(buffer); i += 1 {
		if i < 128 {
			c.Check(buffer[i], Equals, byte(i))
		} else {
			c.Check(buffer[i], Equals, byte(0))
		}
	}
}

func HelperWrite96andCheck(c *C, buffer []byte, writer io.Writer, slices chan nextSlice) {
	out := make([]byte, 96)
	for i := 0; i < 96; i += 1 {
		out[i] = byte(i / 2)
	}
	writer.Write(out)
	s1 := <-slices
	c.Check(len(s1.slice), Equals, 96)
	c.Check(s1.reader_error, Equals, nil)
	for i := 0; i < 96; i += 1 {
		c.Check(s1.slice[i], Equals, byte(i/2))
	}
	for i := 0; i < len(buffer); i += 1 {
		if i < 128 {
			c.Check(buffer[i], Equals, byte(i))
		} else if i < (128 + 96) {
			c.Check(buffer[i], Equals, byte((i-128)/2))
		} else {
			c.Check(buffer[i], Equals, byte(0))
		}
	}
}

func ReadIntoBufferHelper(c *C, bufsize int) {
	buffer := make([]byte, bufsize)

	reader, writer := io.Pipe()
	slices := make(chan nextSlice)

	go readIntoBuffer(buffer, reader, slices)

	HelperWrite128andCheck(c, buffer, writer, slices)
	HelperWrite96andCheck(c, buffer, writer, slices)

	writer.Close()
	s1 := <-slices
	c.Check(len(s1.slice), Equals, 0)
	c.Check(s1.reader_error, Equals, io.EOF)
}

func (s *StandaloneSuite) TestReadIntoShortBuffer(c *C) {
	buffer := make([]byte, 223)
	reader, writer := io.Pipe()
	slices := make(chan nextSlice)

	go readIntoBuffer(buffer, reader, slices)

	HelperWrite128andCheck(c, buffer, writer, slices)

	out := make([]byte, 96)
	for i := 0; i < 96; i += 1 {
		out[i] = byte(i / 2)
	}

	// Write will deadlock because it can't write all the data, so
	// spin it off to a goroutine
	go writer.Write(out)
	s1 := <-slices

	c.Check(len(s1.slice), Equals, 95)
	c.Check(s1.reader_error, Equals, nil)
	for i := 0; i < 95; i += 1 {
		c.Check(s1.slice[i], Equals, byte(i/2))
	}
	for i := 0; i < len(buffer); i += 1 {
		if i < 128 {
			c.Check(buffer[i], Equals, byte(i))
		} else if i < (128 + 95) {
			c.Check(buffer[i], Equals, byte((i-128)/2))
		} else {
			c.Check(buffer[i], Equals, byte(0))
		}
	}

	writer.Close()
	s1 = <-slices
	c.Check(len(s1.slice), Equals, 0)
	c.Check(s1.reader_error, Equals, io.ErrShortBuffer)
}

func (s *StandaloneSuite) TestTransfer(c *C) {
	reader, writer := io.Pipe()

	tr := AsyncStreamFromReader(512, reader)

	br1 := tr.MakeStreamReader()
	out := make([]byte, 128)

	{
		// Write some data, and read into a buffer shorter than
		// available data
		for i := 0; i < 128; i += 1 {
			out[i] = byte(i)
		}

		writer.Write(out[:100])

		in := make([]byte, 64)
		n, err := br1.Read(in)

		c.Check(n, Equals, 64)
		c.Check(err, Equals, nil)

		for i := 0; i < 64; i += 1 {
			c.Check(in[i], Equals, out[i])
		}
	}

	{
		// Write some more data, and read into buffer longer than
		// available data
		in := make([]byte, 64)
		n, err := br1.Read(in)
		c.Check(n, Equals, 36)
		c.Check(err, Equals, nil)

		for i := 0; i < 36; i += 1 {
			c.Check(in[i], Equals, out[64+i])
		}

	}

	{
		// Test read before write
		type Rd struct {
			n   int
			err error
		}
		rd := make(chan Rd)
		in := make([]byte, 64)

		go func() {
			n, err := br1.Read(in)
			rd <- Rd{n, err}
		}()

		time.Sleep(100 * time.Millisecond)
		writer.Write(out[100:])

		got := <-rd

		c.Check(got.n, Equals, 28)
		c.Check(got.err, Equals, nil)

		for i := 0; i < 28; i += 1 {
			c.Check(in[i], Equals, out[100+i])
		}
	}

	br2 := tr.MakeStreamReader()
	{
		// Test 'catch up' reader
		in := make([]byte, 256)
		n, err := br2.Read(in)

		c.Check(n, Equals, 128)
		c.Check(err, Equals, nil)

		for i := 0; i < 128; i += 1 {
			c.Check(in[i], Equals, out[i])
		}
	}

	{
		// Test closing the reader
		writer.Close()

		in := make([]byte, 256)
		n1, err1 := br1.Read(in)
		n2, err2 := br2.Read(in)
		c.Check(n1, Equals, 0)
		c.Check(err1, Equals, io.EOF)
		c.Check(n2, Equals, 0)
		c.Check(err2, Equals, io.EOF)
	}

	{
		// Test 'catch up' reader after closing
		br3 := tr.MakeStreamReader()
		in := make([]byte, 256)
		n, err := br3.Read(in)

		c.Check(n, Equals, 128)
		c.Check(err, Equals, nil)

		for i := 0; i < 128; i += 1 {
			c.Check(in[i], Equals, out[i])
		}

		n, err = br3.Read(in)

		c.Check(n, Equals, 0)
		c.Check(err, Equals, io.EOF)
	}
}

func (s *StandaloneSuite) TestTransferShortBuffer(c *C) {
	reader, writer := io.Pipe()

	tr := AsyncStreamFromReader(100, reader)
	defer tr.Close()

	sr := tr.MakeStreamReader()
	defer sr.Close()

	out := make([]byte, 101)
	go writer.Write(out)

	n, err := sr.Read(out)
	c.Check(n, Equals, 100)

	n, err = sr.Read(out)
	c.Check(n, Equals, 0)
	c.Check(err, Equals, io.ErrShortBuffer)
}

func (s *StandaloneSuite) TestTransferFromBuffer(c *C) {
	// Buffer for reads from 'r'
	buffer := make([]byte, 100)
	for i := 0; i < 100; i += 1 {
		buffer[i] = byte(i)
	}

	tr := AsyncStreamFromSlice(buffer)

	br1 := tr.MakeStreamReader()

	in := make([]byte, 64)
	{
		n, err := br1.Read(in)

		c.Check(n, Equals, 64)
		c.Check(err, Equals, nil)

		for i := 0; i < 64; i += 1 {
			c.Check(in[i], Equals, buffer[i])
		}
	}
	{
		n, err := br1.Read(in)

		c.Check(n, Equals, 36)
		c.Check(err, Equals, nil)

		for i := 0; i < 36; i += 1 {
			c.Check(in[i], Equals, buffer[64+i])
		}
	}
	{
		n, err := br1.Read(in)

		c.Check(n, Equals, 0)
		c.Check(err, Equals, io.EOF)
	}
}

func (s *StandaloneSuite) TestTransferIoCopy(c *C) {
	// Buffer for reads from 'r'
	buffer := make([]byte, 100)
	for i := 0; i < 100; i += 1 {
		buffer[i] = byte(i)
	}

	tr := AsyncStreamFromSlice(buffer)
	defer tr.Close()

	br1 := tr.MakeStreamReader()
	defer br1.Close()

	reader, writer := io.Pipe()

	go func() {
		p := make([]byte, 100)
		n, err := reader.Read(p)
		c.Check(n, Equals, 100)
		c.Check(err, Equals, nil)
		c.Check(p, DeepEquals, buffer)
	}()

	io.Copy(writer, br1)
}

func (s *StandaloneSuite) TestManyReaders(c *C) {
	reader, writer := io.Pipe()

	tr := AsyncStreamFromReader(512, reader)
	defer tr.Close()

	sr := tr.MakeStreamReader()
	go func() {
		time.Sleep(100 * time.Millisecond)
		sr.Close()
	}()

	for i := 0; i < 200; i += 1 {
		go func() {
			br1 := tr.MakeStreamReader()
			defer br1.Close()

			p := make([]byte, 3)
			n, err := br1.Read(p)
			c.Check(n, Equals, 3)
			c.Check(p[0:3], DeepEquals, []byte("foo"))

			n, err = br1.Read(p)
			c.Check(n, Equals, 3)
			c.Check(p[0:3], DeepEquals, []byte("bar"))

			n, err = br1.Read(p)
			c.Check(n, Equals, 3)
			c.Check(p[0:3], DeepEquals, []byte("baz"))

			n, err = br1.Read(p)
			c.Check(n, Equals, 0)
			c.Check(err, Equals, io.EOF)
		}()
	}

	writer.Write([]byte("foo"))
	writer.Write([]byte("bar"))
	writer.Write([]byte("baz"))
	writer.Close()
}
