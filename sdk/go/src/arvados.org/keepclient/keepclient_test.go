package keepclient

import (
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestGetKeepDisks(c *C) {
	k, err := KeepDisks()
	c.Assert(err, Equals, nil)
	c.Assert(len(k), Equals, 2)
	c.Assert(k[0].Hostname, Equals, "localhost")
	c.Assert(k[0].Port, Equals, 25108)
	c.Assert(k[1].Hostname, Equals, "localhost")
	c.Assert(k[1].Port, Equals, 25107)
}
