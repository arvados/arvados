// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
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
	serviceSuite serviceSuite
	token        string
	toDelete     []string
	wg           sync.WaitGroup
	ignoreLogID  int64
}

func (s *v0Suite) SetUpTest(c *check.C) {
	s.serviceSuite.SetUpTest(c)
	s.serviceSuite.start(c)

	s.token = arvadostest.ActiveToken
	s.ignoreLogID = s.lastLogID(c)
}

func (s *v0Suite) TearDownTest(c *check.C) {
	s.wg.Wait()
	s.serviceSuite.TearDownTest(c)
}

func (s *v0Suite) TearDownSuite(c *check.C) {
	s.deleteTestObjects(c)
}

func (s *v0Suite) deleteTestObjects(c *check.C) {
	ac := arvados.NewClientFromEnv()
	ac.AuthToken = arvadostest.AdminToken
	for _, path := range s.toDelete {
		err := ac.RequestAndDecode(nil, "DELETE", path, nil, nil)
		if err != nil {
			panic(err)
		}
	}
	s.toDelete = nil
}

func (s *v0Suite) TestFilters(c *check.C) {
	conn, r, w := s.testClient()
	defer conn.Close()

	cmd := func(method, eventType string, status int) {
		c.Check(w.Encode(map[string]interface{}{
			"method":  method,
			"filters": [][]interface{}{{"event_type", "in", []string{eventType}}},
		}), check.IsNil)
		s.expectStatus(c, r, status)
	}
	cmd("subscribe", "update", 200)
	cmd("subscribe", "update", 200)
	cmd("subscribe", "create", 200)
	cmd("subscribe", "update", 200)
	cmd("unsubscribe", "blip", 400)
	cmd("unsubscribe", "create", 200)
	cmd("unsubscribe", "update", 200)

	go s.emitEvents(nil)
	lg := s.expectLog(c, r)
	c.Check(lg.EventType, check.Equals, "update")

	cmd("unsubscribe", "update", 200)
	cmd("unsubscribe", "update", 200)
	cmd("unsubscribe", "update", 400)
}

func (s *v0Suite) TestLastLogID(c *check.C) {
	lastID := s.lastLogID(c)

	checkLogs := func(r *json.Decoder, uuid string) {
		for _, etype := range []string{"create", "blip", "update"} {
			lg := s.expectLog(c, r)
			for lg.ObjectUUID != uuid {
				lg = s.expectLog(c, r)
			}
			c.Check(lg.EventType, check.Equals, etype)
		}
	}

	// Connecting connEarly (before sending the early events) lets
	// us confirm all of the "early" events have already passed
	// through the server.
	connEarly, rEarly, wEarly := s.testClient()
	defer connEarly.Close()
	c.Check(wEarly.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, rEarly, 200)

	// Send the early events.
	uuidChan := make(chan string, 1)
	s.emitEvents(uuidChan)
	uuidEarly := <-uuidChan

	// Wait for the early events to pass through.
	checkLogs(rEarly, uuidEarly)

	// Connect the client that wants to get old events via
	// last_log_id.
	conn, r, w := s.testClient()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method":      "subscribe",
		"last_log_id": lastID,
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	checkLogs(r, uuidEarly)
	s.emitEvents(uuidChan)
	checkLogs(r, <-uuidChan)
}

func (s *v0Suite) TestPermission(c *check.C) {
	conn, r, w := s.testClient()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	uuidChan := make(chan string, 2)
	go func() {
		s.token = arvadostest.AdminToken
		s.emitEvents(uuidChan)
		s.token = arvadostest.ActiveToken
		s.emitEvents(uuidChan)
	}()

	wrongUUID := <-uuidChan
	rightUUID := <-uuidChan
	lg := s.expectLog(c, r)
	for lg.ObjectUUID != rightUUID {
		c.Check(lg.ObjectUUID, check.Not(check.Equals), wrongUUID)
		lg = s.expectLog(c, r)
	}
}

// Two users create private objects; admin deletes both objects; each
// user receives a "delete" event for their own object (not for the
// other user's object).
func (s *v0Suite) TestEventTypeDelete(c *check.C) {
	clients := []struct {
		token string
		uuid  string
		conn  *websocket.Conn
		r     *json.Decoder
		w     *json.Encoder
	}{{token: arvadostest.ActiveToken}, {token: arvadostest.SpectatorToken}}
	for i := range clients {
		uuidChan := make(chan string, 1)
		s.token = clients[i].token
		s.emitEvents(uuidChan)
		clients[i].uuid = <-uuidChan
		clients[i].conn, clients[i].r, clients[i].w = s.testClient()

		c.Check(clients[i].w.Encode(map[string]interface{}{
			"method": "subscribe",
		}), check.IsNil)
		s.expectStatus(c, clients[i].r, 200)
	}

	s.ignoreLogID = s.lastLogID(c)
	s.deleteTestObjects(c)

	for _, client := range clients {
		lg := s.expectLog(c, client.r)
		c.Check(lg.ObjectUUID, check.Equals, client.uuid)
		c.Check(lg.EventType, check.Equals, "delete")
	}
}

// Trashing/deleting a collection produces an "update" event with
// properties["new_attributes"]["is_trashed"] == true.
func (s *v0Suite) TestTrashedCollection(c *check.C) {
	ac := arvados.NewClientFromEnv()
	ac.AuthToken = s.token

	var coll arvados.Collection
	err := ac.RequestAndDecode(&coll, "POST", "arvados/v1/collections", s.jsonBody("collection", `{"manifest_text":""}`), map[string]interface{}{"ensure_unique_name": true})
	c.Assert(err, check.IsNil)
	s.ignoreLogID = s.lastLogID(c)

	conn, r, w := s.testClient()
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	err = ac.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)
	c.Assert(err, check.IsNil)

	lg := s.expectLog(c, r)
	c.Check(lg.ObjectUUID, check.Equals, coll.UUID)
	c.Check(lg.EventType, check.Equals, "update")
	c.Check(lg.Properties["old_attributes"].(map[string]interface{})["is_trashed"], check.Equals, false)
	c.Check(lg.Properties["new_attributes"].(map[string]interface{})["is_trashed"], check.Equals, true)
}

func (s *v0Suite) TestSendBadJSON(c *check.C) {
	conn, r, w := s.testClient()
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
	conn, r, w := s.testClient()
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
		for lg.ObjectUUID != uuid {
			lg = s.expectLog(c, r)
		}
		c.Check(lg.EventType, check.Equals, etype)
	}
}

// Generate some events by creating and updating a workflow object,
// and creating a custom log entry (event_type="blip") about the newly
// created workflow. If uuidChan is not nil, send the new workflow
// UUID to uuidChan as soon as it's known.
func (s *v0Suite) emitEvents(uuidChan chan<- string) {
	s.wg.Add(1)
	defer s.wg.Done()

	ac := arvados.NewClientFromEnv()
	ac.AuthToken = s.token
	wf := &arvados.Workflow{
		Name: "ws_test",
	}
	err := ac.RequestAndDecode(wf, "POST", "arvados/v1/workflows", s.jsonBody("workflow", `{"name":"ws_test"}`), map[string]interface{}{"ensure_unique_name": true})
	if err != nil {
		panic(err)
	}
	if uuidChan != nil {
		uuidChan <- wf.UUID
	}
	lg := &arvados.Log{}
	err = ac.RequestAndDecode(lg, "POST", "arvados/v1/logs", s.jsonBody("log", map[string]interface{}{
		"object_uuid": wf.UUID,
		"event_type":  "blip",
		"properties": map[string]interface{}{
			"beep": "boop",
		},
	}), nil)
	if err != nil {
		panic(err)
	}
	err = ac.RequestAndDecode(wf, "PUT", "arvados/v1/workflows/"+wf.UUID, s.jsonBody("workflow", `{"name":"ws_test"}`), nil)
	if err != nil {
		panic(err)
	}
	s.toDelete = append(s.toDelete, "arvados/v1/workflows/"+wf.UUID, "arvados/v1/logs/"+lg.UUID)
}

func (s *v0Suite) jsonBody(rscName string, ob interface{}) io.Reader {
	val, ok := ob.(string)
	if !ok {
		j, err := json.Marshal(ob)
		if err != nil {
			panic(err)
		}
		val = string(j)
	}
	v := url.Values{}
	v[rscName] = []string{val}
	return bytes.NewBufferString(v.Encode())
}

func (s *v0Suite) expectStatus(c *check.C, r *json.Decoder, status int) {
	msg := map[string]interface{}{}
	c.Check(r.Decode(&msg), check.IsNil)
	c.Check(int(msg["status"].(float64)), check.Equals, status)
}

func (s *v0Suite) expectLog(c *check.C, r *json.Decoder) *arvados.Log {
	lg := &arvados.Log{}
	ok := make(chan struct{})
	go func() {
		defer close(ok)
		for lg.ID <= s.ignoreLogID {
			c.Assert(r.Decode(lg), check.IsNil)
		}
	}()
	select {
	case <-time.After(10 * time.Second):
		c.Error("timed out")
		c.FailNow()
		return lg
	case <-ok:
		return lg
	}
}

func (s *v0Suite) testClient() (*websocket.Conn, *json.Decoder, *json.Encoder) {
	srv := s.serviceSuite.srv
	conn, err := websocket.Dial(strings.Replace(srv.URL, "http", "ws", 1)+"/websocket?api_token="+s.token, "", srv.URL)
	if err != nil {
		panic(err)
	}
	w := json.NewEncoder(conn)
	r := json.NewDecoder(conn)
	return conn, r, w
}

func (s *v0Suite) lastLogID(c *check.C) int64 {
	var lastID int64
	c.Assert(testDB().QueryRow(`SELECT MAX(id) FROM logs`).Scan(&lastID), check.IsNil)
	return lastID
}
