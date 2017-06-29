// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"golang.org/x/net/websocket"
	check "gopkg.in/check.v1"
)

func init() {
	if os.Getenv("ARVADOS_DEBUG") != "" {
		ctxlog.SetLevel("debug")
	}
}

var _ = check.Suite(&v0Suite{})

type v0Suite struct {
	serverSuite serverSuite
	token       string
	toDelete    []string
}

func (s *v0Suite) SetUpTest(c *check.C) {
	s.serverSuite.SetUpTest(c)
	s.token = arvadostest.ActiveToken
}

func (s *v0Suite) TearDownSuite(c *check.C) {
	ac := arvados.NewClientFromEnv()
	ac.AuthToken = arvadostest.AdminToken
	for _, path := range s.toDelete {
		err := ac.RequestAndDecode(nil, "DELETE", path, nil, nil)
		if err != nil {
			panic(err)
		}
	}
}

func (s *v0Suite) TestFilters(c *check.C) {
	srv, conn, r, w := s.testClient()
	defer srv.Close()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method":  "subscribe",
		"filters": [][]interface{}{{"event_type", "in", []string{"update"}}},
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	go s.emitEvents(nil)
	lg := s.expectLog(c, r)
	c.Check(lg.EventType, check.Equals, "update")
}

func (s *v0Suite) TestLastLogID(c *check.C) {
	var lastID uint64
	c.Assert(testDB().QueryRow(`SELECT MAX(id) FROM logs`).Scan(&lastID), check.IsNil)

	srv, conn, r, w := s.testClient()
	defer srv.Close()
	defer conn.Close()

	uuidChan := make(chan string, 2)
	s.emitEvents(uuidChan)

	c.Check(w.Encode(map[string]interface{}{
		"method":      "subscribe",
		"last_log_id": lastID,
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	go func() {
		s.emitEvents(uuidChan)
		close(uuidChan)
	}()

	done := make(chan bool)
	go func() {
		for uuid := range uuidChan {
			for _, etype := range []string{"create", "blip", "update"} {
				lg := s.expectLog(c, r)
				c.Check(lg.ObjectUUID, check.Equals, uuid)
				c.Check(lg.EventType, check.Equals, etype)
			}
		}
		close(done)
	}()

	select {
	case <-time.After(10 * time.Second):
		c.Fatal("timeout")
	case <-done:
	}
}

func (s *v0Suite) TestPermission(c *check.C) {
	srv, conn, r, w := s.testClient()
	defer srv.Close()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	uuidChan := make(chan string, 1)
	go func() {
		s.token = arvadostest.AdminToken
		s.emitEvents(nil)
		s.token = arvadostest.ActiveToken
		s.emitEvents(uuidChan)
	}()

	lg := s.expectLog(c, r)
	c.Check(lg.ObjectUUID, check.Equals, <-uuidChan)
}

func (s *v0Suite) TestSendBadJSON(c *check.C) {
	srv, conn, r, w := s.testClient()
	defer srv.Close()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	_, err := fmt.Fprint(conn, "^]beep\n")
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 400)

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)
}

func (s *v0Suite) TestSubscribe(c *check.C) {
	srv, conn, r, w := s.testClient()
	defer srv.Close()
	defer conn.Close()

	s.emitEvents(nil)

	err := w.Encode(map[string]interface{}{"21": 12})
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 400)

	err = w.Encode(map[string]interface{}{"method": "subscribe", "filters": []string{}})
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 200)

	uuidChan := make(chan string, 1)
	go s.emitEvents(uuidChan)
	uuid := <-uuidChan

	for _, etype := range []string{"create", "blip", "update"} {
		lg := s.expectLog(c, r)
		c.Check(lg.ObjectUUID, check.Equals, uuid)
		c.Check(lg.EventType, check.Equals, etype)
	}
}

// Generate some events by creating and updating a workflow object,
// and creating a custom log entry (event_type="blip") about the newly
// created workflow. If uuidChan is not nil, send the new workflow
// UUID to uuidChan as soon as it's known.
func (s *v0Suite) emitEvents(uuidChan chan<- string) {
	ac := arvados.NewClientFromEnv()
	ac.AuthToken = s.token
	wf := &arvados.Workflow{
		Name: "ws_test",
	}
	err := ac.RequestAndDecode(wf, "POST", "arvados/v1/workflows", s.jsonBody("workflow", wf), map[string]interface{}{"ensure_unique_name": true})
	if err != nil {
		panic(err)
	}
	if uuidChan != nil {
		uuidChan <- wf.UUID
	}
	lg := &arvados.Log{}
	err = ac.RequestAndDecode(lg, "POST", "arvados/v1/logs", s.jsonBody("log", &arvados.Log{
		ObjectUUID: wf.UUID,
		EventType:  "blip",
		Properties: map[string]interface{}{
			"beep": "boop",
		},
	}), nil)
	if err != nil {
		panic(err)
	}
	err = ac.RequestAndDecode(wf, "PUT", "arvados/v1/workflows/"+wf.UUID, s.jsonBody("workflow", wf), nil)
	if err != nil {
		panic(err)
	}
	s.toDelete = append(s.toDelete, "arvados/v1/workflows/"+wf.UUID, "arvados/v1/logs/"+lg.UUID)
}

func (s *v0Suite) jsonBody(rscName string, ob interface{}) io.Reader {
	j, err := json.Marshal(ob)
	if err != nil {
		panic(err)
	}
	v := url.Values{}
	v[rscName] = []string{string(j)}
	return bytes.NewBufferString(v.Encode())
}

func (s *v0Suite) expectStatus(c *check.C, r *json.Decoder, status int) {
	msg := map[string]interface{}{}
	c.Check(r.Decode(&msg), check.IsNil)
	c.Check(int(msg["status"].(float64)), check.Equals, status)
}

func (s *v0Suite) expectLog(c *check.C, r *json.Decoder) *arvados.Log {
	lg := &arvados.Log{}
	c.Check(r.Decode(lg), check.IsNil)
	return lg
}

func (s *v0Suite) testClient() (*server, *websocket.Conn, *json.Decoder, *json.Encoder) {
	go s.serverSuite.srv.Run()
	s.serverSuite.srv.WaitReady()
	srv := s.serverSuite.srv
	conn, err := websocket.Dial("ws://"+srv.listener.Addr().String()+"/websocket?api_token="+s.token, "", "http://"+srv.listener.Addr().String())
	if err != nil {
		panic(err)
	}
	w := json.NewEncoder(conn)
	r := json.NewDecoder(conn)
	return srv, conn, r, w
}
