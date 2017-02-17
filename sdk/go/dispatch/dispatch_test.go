package dispatch

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = Suite(&DispatchTestSuite{})

type DispatchTestSuite struct{}

func (s *DispatchTestSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
}

func (s *DispatchTestSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
}

func (s *DispatchTestSuite) TestTrackContainer(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	d := &Dispatcher{Arv: arv}
	d.trackers = make(map[string]*runTracker)

	d.TrackContainer(arvadostest.QueuedContainerUuid)
	_, tracking := d.trackers[arvadostest.QueuedContainerUuid]
	c.Assert(tracking, Equals, true)
}
