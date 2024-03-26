// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

func (s *routerSuite) TestPullList_Execute(c *C) {
	remotecluster := testCluster(c)
	remotecluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-rrrrrrrrrrrrrrr": {Replication: 1, Driver: "stub"},
	}
	remoterouter, cancel := testRouter(c, remotecluster, nil)
	defer cancel()
	remoteserver := httptest.NewServer(remoterouter)
	defer remoteserver.Close()

	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	executePullList := func(pullList []PullListItem) string {
		var logbuf bytes.Buffer
		logger := logrus.New()
		logger.Out = &logbuf
		router.keepstore.logger = logger

		listjson, err := json.Marshal(pullList)
		c.Assert(err, IsNil)
		resp := call(router, "PUT", "http://example/pull", s.cluster.SystemRootToken, listjson, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		for {
			router.puller.cond.L.Lock()
			todolen := len(router.puller.todo)
			router.puller.cond.L.Unlock()
			if todolen == 0 && router.puller.inprogress.Load() == 0 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		return logbuf.String()
	}

	newRemoteBlock := func(datastring string) string {
		data := []byte(datastring)
		hash := fmt.Sprintf("%x", md5.Sum(data))
		locator := fmt.Sprintf("%s+%d", hash, len(data))
		_, err := remoterouter.keepstore.BlockWrite(context.Background(), arvados.BlockWriteOptions{
			Hash: hash,
			Data: data,
		})
		c.Assert(err, IsNil)
		return locator
	}

	mounts := append([]*mount(nil), router.keepstore.mountsR...)
	sort.Slice(mounts, func(i, j int) bool { return mounts[i].UUID < mounts[j].UUID })
	var vols []*stubVolume
	for _, mount := range mounts {
		vols = append(vols, mount.volume.(*stubVolume))
	}

	ctx := authContext(arvadostest.ActiveTokenV2)

	locator := newRemoteBlock("pull available block to unspecified volume")
	executePullList([]PullListItem{{
		Locator: locator,
		Servers: []string{remoteserver.URL}}})
	_, err := router.keepstore.BlockRead(ctx, arvados.BlockReadOptions{
		Locator: router.keepstore.signLocator(arvadostest.ActiveTokenV2, locator),
		WriteTo: io.Discard})
	c.Check(err, IsNil)

	locator0 := newRemoteBlock("pull available block to specified volume 0")
	locator1 := newRemoteBlock("pull available block to specified volume 1")
	executePullList([]PullListItem{
		{
			Locator:   locator0,
			Servers:   []string{remoteserver.URL},
			MountUUID: vols[0].params.UUID},
		{
			Locator:   locator1,
			Servers:   []string{remoteserver.URL},
			MountUUID: vols[1].params.UUID}})
	c.Check(vols[0].data[locator0[:32]].data, NotNil)
	c.Check(vols[1].data[locator1[:32]].data, NotNil)

	locator = fooHash + "+3"
	logs := executePullList([]PullListItem{{
		Locator: locator,
		Servers: []string{remoteserver.URL}}})
	c.Check(logs, Matches, ".*error pulling data from remote servers.*Block not found.*locator=acbd.*\n")

	locator = fooHash + "+3"
	logs = executePullList([]PullListItem{{
		Locator: locator,
		Servers: []string{"http://0.0.0.0:9/"}}})
	c.Check(logs, Matches, ".*error pulling data from remote servers.*connection refused.*locator=acbd.*\n")

	locator = newRemoteBlock("log error writing to local volume")
	vols[0].blockWrite = func(context.Context, string, []byte) error { return errors.New("test error") }
	vols[1].blockWrite = vols[0].blockWrite
	logs = executePullList([]PullListItem{{
		Locator: locator,
		Servers: []string{remoteserver.URL}}})
	c.Check(logs, Matches, ".*error writing data to zzzzz-nyw5e-.*error=\"test error\".*locator=.*\n")
	vols[0].blockWrite = nil
	vols[1].blockWrite = nil

	locator = newRemoteBlock("log error when destination mount does not exist")
	logs = executePullList([]PullListItem{{
		Locator:   locator,
		Servers:   []string{remoteserver.URL},
		MountUUID: "bogus-mount-uuid"}})
	c.Check(logs, Matches, ".*ignoring pull list entry for nonexistent mount bogus-mount-uuid.*locator=.*\n")

	logs = executePullList([]PullListItem{})
	c.Logf("%s", logs)
}
