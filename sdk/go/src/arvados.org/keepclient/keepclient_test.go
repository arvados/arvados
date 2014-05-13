package keepclient

import (
	"flag"
	//"fmt"
	. "gopkg.in/check.v1"
	"io"
	//"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Gocheck boilerplate
func Test(t *testing.T) { TestingT(t) }

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&StandaloneSuite{})

var no_server = flag.Bool("no-server", false, "Skip 'ServerRequireSuite'")

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Standalone tests
type StandaloneSuite struct{}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	if *no_server {
		c.Skip("Skipping tests that require server")
	} else {
		os.Chdir(os.ExpandEnv("$GOPATH../python"))
		exec.Command("python", "run_test_server.py", "start").Run()
		exec.Command("python", "run_test_server.py", "start_keep").Run()
	}
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	os.Chdir(os.ExpandEnv("$GOPATH../python"))
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func (s *ServerRequiredSuite) TestInit(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3001")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "")

	kc, err := MakeKeepClient()
	c.Assert(kc.ApiServer, Equals, "localhost:3001")
	c.Assert(kc.ApiToken, Equals, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	c.Assert(kc.ApiInsecure, Equals, false)

	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	kc, err = MakeKeepClient()
	c.Assert(kc.ApiServer, Equals, "localhost:3001")
	c.Assert(kc.ApiToken, Equals, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	c.Assert(kc.ApiInsecure, Equals, true)

	c.Assert(err, Equals, nil)
	c.Assert(len(kc.Service_roots), Equals, 2)
	c.Assert(kc.Service_roots[0], Equals, "http://localhost:25107")
	c.Assert(kc.Service_roots[1], Equals, "http://localhost:25108")
}

func (s *StandaloneSuite) TestShuffleServiceRoots(c *C) {
	kc := KeepClient{Service_roots: []string{"http://localhost:25107", "http://localhost:25108", "http://localhost:25109", "http://localhost:25110", "http://localhost:25111", "http://localhost:25112", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25116", "http://localhost:25117", "http://localhost:25118", "http://localhost:25119", "http://localhost:25120", "http://localhost:25121", "http://localhost:25122", "http://localhost:25123"}}

	// "foo" acbd18db4cc2f85cedef654fccc4a4d8
	foo_shuffle := []string{"http://localhost:25116", "http://localhost:25120", "http://localhost:25119", "http://localhost:25122", "http://localhost:25108", "http://localhost:25114", "http://localhost:25112", "http://localhost:25107", "http://localhost:25118", "http://localhost:25111", "http://localhost:25113", "http://localhost:25121", "http://localhost:25110", "http://localhost:25117", "http://localhost:25109", "http://localhost:25115", "http://localhost:25123"}
	c.Check(kc.ShuffledServiceRoots("acbd18db4cc2f85cedef654fccc4a4d8"), DeepEquals, foo_shuffle)

	// "bar" 37b51d194a7513e45b56f6524f2d51f2
	bar_shuffle := []string{"http://localhost:25108", "http://localhost:25112", "http://localhost:25119", "http://localhost:25107", "http://localhost:25110", "http://localhost:25116", "http://localhost:25122", "http://localhost:25120", "http://localhost:25121", "http://localhost:25117", "http://localhost:25111", "http://localhost:25123", "http://localhost:25118", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25109"}
	c.Check(kc.ShuffledServiceRoots("37b51d194a7513e45b56f6524f2d51f2"), DeepEquals, bar_shuffle)
}

func ReadIntoBufferHelper(c *C, bufsize int) {
	buffer := make([]byte, bufsize)

	reader, writer := io.Pipe()
	slices := make(chan ReaderSlice)

	go ReadIntoBuffer(buffer, reader, slices)

	{
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
	{
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
	{
		writer.Close()
		s1 := <-slices
		c.Check(len(s1.slice), Equals, 0)
		c.Check(s1.reader_error, Equals, io.EOF)
	}
}

func (s *StandaloneSuite) TestReadIntoBuffer(c *C) {
	ReadIntoBufferHelper(c, 512)
	ReadIntoBufferHelper(c, 225)
	ReadIntoBufferHelper(c, 224)
}

func (s *StandaloneSuite) TestReadIntoShortBuffer(c *C) {
	buffer := make([]byte, 223)
	reader, writer := io.Pipe()
	slices := make(chan ReaderSlice)

	go ReadIntoBuffer(buffer, reader, slices)

	{
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
	{
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
	}
	{
		writer.Close()
		s1 := <-slices
		c.Check(len(s1.slice), Equals, 0)
		c.Check(s1.reader_error, Equals, io.ErrShortBuffer)
	}

}

func (s *StandaloneSuite) TestTransfer(c *C) {
	reader, writer := io.Pipe()

	// Buffer for reads from 'r'
	buffer := make([]byte, 512)

	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Reporting reader error states
	reader_status := make(chan error)

	go Transfer(buffer, reader, requests, reader_status)

	br1 := MakeBufferReader(requests)
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

	br2 := MakeBufferReader(requests)
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
		status := <-reader_status
		c.Check(status, Equals, io.EOF)

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
		br3 := MakeBufferReader(requests)
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

	// Buffer for reads from 'r'
	buffer := make([]byte, 100)

	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Reporting reader error states
	reader_status := make(chan error)

	go Transfer(buffer, reader, requests, reader_status)

	out := make([]byte, 101)
	go writer.Write(out)

	status := <-reader_status
	c.Check(status, Equals, io.ErrShortBuffer)
}
