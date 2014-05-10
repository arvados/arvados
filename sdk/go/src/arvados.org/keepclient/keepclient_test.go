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
	sr, err := KeepDisks()
	c.Assert(err, Equals, nil)
	c.Assert(len(sr), Equals, 2)
	c.Assert(sr[0], Equals, "http://localhost:25107")
	c.Assert(sr[1], Equals, "http://localhost:25108")

}
