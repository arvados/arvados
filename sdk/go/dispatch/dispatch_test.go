// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package dispatch

import (
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = Suite(&suite{})

type suite struct{}

func (s *suite) TestTrackContainer(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	arv.ApiToken = arvadostest.Dispatch1Token

	done := make(chan bool, 1)
	time.AfterFunc(10*time.Second, func() { done <- false })
	d := &Dispatcher{
		Arv: arv,
		RunContainer: func(dsp *Dispatcher, ctr arvados.Container, status <-chan arvados.Container) error {
			for ctr := range status {
				c.Logf("%#v", ctr)
			}
			done <- true
			return nil
		},
	}
	d.TrackContainer(arvadostest.QueuedContainerUUID)
	c.Assert(<-done, Equals, true)
}
