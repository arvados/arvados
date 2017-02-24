package dispatch

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"os/exec"

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

	runContainer := func(d *Dispatcher, ctr arvados.Container) *exec.Cmd { return exec.Command("echo") }
	d := &Dispatcher{Arv: arv, RunContainer: func(dsp *Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
		go runContainer(dsp, ctr)
	}}
	d.trackers = make(map[string]*runTracker)

	d.TrackContainer(arvadostest.QueuedContainerUuid)
	_, tracking := d.trackers[arvadostest.QueuedContainerUuid]
	c.Assert(tracking, Equals, true)
}
