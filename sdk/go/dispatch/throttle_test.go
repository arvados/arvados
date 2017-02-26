package dispatch

import (
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ThrottleTestSuite{})

type ThrottleTestSuite struct{}

func (*ThrottleTestSuite) TestThrottle(c *check.C) {
	uuid := "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

	t := throttle{}
	c.Check(t.Check(uuid), check.Equals, true)
	c.Check(t.Check(uuid), check.Equals, true)

	t = throttle{hold: time.Nanosecond}
	c.Check(t.Check(uuid), check.Equals, true)
	time.Sleep(time.Microsecond)
	c.Check(t.Check(uuid), check.Equals, true)

	t = throttle{hold: time.Minute}
	c.Check(t.Check(uuid), check.Equals, true)
	c.Check(t.Check(uuid), check.Equals, false)
	c.Check(t.Check(uuid), check.Equals, false)
	t.seen[uuid].last = time.Now().Add(-time.Hour)
	c.Check(t.Check(uuid), check.Equals, true)
	c.Check(t.Check(uuid), check.Equals, false)
}
