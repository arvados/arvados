// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
		c.Check(err, check.IsNil)
	}
	s.toDelete = nil
}

func (s *v0Suite) TestFilters(c *check.C) {
	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
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

	go s.emitEvents(c, nil, nil)
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
	connEarly, rEarly, wEarly, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer connEarly.Close()
	c.Check(wEarly.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, rEarly, 200)

	// Send the early events.
	uuidChan := make(chan string, 1)
	s.emitEvents(c, uuidChan, nil)
	uuidEarly := <-uuidChan

	// Wait for the early events to pass through.
	checkLogs(rEarly, uuidEarly)

	// Connect the client that wants to get old events via
	// last_log_id.
	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method":      "subscribe",
		"last_log_id": lastID,
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	checkLogs(r, uuidEarly)
	s.emitEvents(c, uuidChan, nil)
	checkLogs(r, <-uuidChan)
}

func (s *v0Suite) TestPermission(c *check.C) {
	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	uuidChan := make(chan string, 2)
	go func() {
		s.token = arvadostest.AdminToken
		s.emitEvents(c, uuidChan, nil)
		s.token = arvadostest.ActiveToken
		s.emitEvents(c, uuidChan, nil)
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
		s.emitEvents(c, uuidChan, nil)
		clients[i].uuid = <-uuidChan

		var err error
		clients[i].conn, clients[i].r, clients[i].w, err = s.testClient()
		c.Assert(err, check.IsNil)

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

func (s *v0Suite) TestEventPropertiesFields(c *check.C) {
	ac := arvados.NewClientFromEnv()
	ac.AuthToken = s.token

	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method":  "subscribe",
		"filters": [][]string{{"object_uuid", "=", arvadostest.RunningContainerUUID}},
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	err = ac.RequestAndDecode(nil, "POST", "arvados/v1/logs", s.jsonBody("log", map[string]interface{}{
		"object_uuid": arvadostest.RunningContainerUUID,
		"event_type":  "update",
		"properties": map[string]interface{}{
			"new_attributes": map[string]interface{}{
				"name":                      "namevalue",
				"requesting_container_uuid": "uuidvalue",
				"state":                     "statevalue",
			},
		},
	}), nil)
	c.Assert(err, check.IsNil)

	lg := s.expectLog(c, r)
	c.Check(lg.ObjectUUID, check.Equals, arvadostest.RunningContainerUUID)
	c.Check(lg.EventType, check.Equals, "update")
	c.Check(lg.Properties["new_attributes"].(map[string]interface{})["requesting_container_uuid"], check.Equals, "uuidvalue")
	c.Check(lg.Properties["new_attributes"].(map[string]interface{})["name"], check.Equals, "namevalue")
	c.Check(lg.Properties["new_attributes"].(map[string]interface{})["state"], check.Equals, "statevalue")
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

	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
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
	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer conn.Close()

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)

	_, err = fmt.Fprint(conn, "^]beep\n")
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 400)

	c.Check(w.Encode(map[string]interface{}{
		"method": "subscribe",
	}), check.IsNil)
	s.expectStatus(c, r, 200)
}

func (s *v0Suite) TestSubscribe(c *check.C) {
	conn, r, w, err := s.testClient()
	c.Assert(err, check.IsNil)
	defer conn.Close()

	s.emitEvents(c, nil, nil)

	err = w.Encode(map[string]interface{}{"21": 12})
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 400)

	err = w.Encode(map[string]interface{}{"method": "subscribe", "filters": []string{}})
	c.Check(err, check.IsNil)
	s.expectStatus(c, r, 200)

	uuidChan := make(chan string, 1)
	go s.emitEvents(c, uuidChan, nil)
	uuid := <-uuidChan

	for _, etype := range []string{"create", "blip", "update"} {
		lg := s.expectLog(c, r)
		for lg.ObjectUUID != uuid {
			lg = s.expectLog(c, r)
		}
		c.Check(lg.EventType, check.Equals, etype)
	}
}

func (s *v0Suite) TestManyEventsAndSubscribers(c *check.C) {
	// Frequent slow listener pings create the conditions for a
	// deadlock issue with the lib/pq example listener usage.
	//
	// Specifically: a lib/pq/example/listen-style event loop can
	// deadlock if enough (~32) server notifications arrive after
	// the event loop decides to call Ping (e.g., while
	// listener.Ping() is waiting for a response from the server,
	// or in the time.Sleep() invoked by testSlowPing).
	//
	// (*ListenerConn)listenerConnLoop() doesn't see the server's
	// ping response until it finishes sending a previous
	// notification through its internal queue to
	// (*Listener)listenerConnLoop(), which is blocked on sending
	// to our Notify channel, which is blocked on waiting for the
	// Ping response.
	defer func(d time.Duration) {
		listenerPingInterval = d
		testSlowPing = false
	}(listenerPingInterval)
	listenerPingInterval = time.Second / 2
	testSlowPing = true
	// Restart the test server in order to get one that uses our
	// test globals.
	s.TearDownTest(c)
	s.SetUpTest(c)

	done := make(chan struct{})
	defer close(done)
	go s.emitEvents(c, nil, done)

	// We will expect to receive at least one event during each
	// one-second interval while the test is running.
	t0 := time.Now()
	seconds := 10
	receivedPerSecond := make([]int64, seconds)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(seconds)*time.Second))
	defer cancel()
	for clientID := 0; clientID < 100; clientID++ {
		clientID := clientID
		go func() {
			for ctx.Err() == nil {
				conn, r, w, err := s.testClient()
				if ctx.Err() != nil {
					return
				}
				c.Assert(err, check.IsNil)
				defer conn.Close()
				err = w.Encode(map[string]interface{}{"method": "subscribe", "filters": []string{}})
				if ctx.Err() != nil {
					return
				}
				c.Check(err, check.IsNil)
				s.expectStatus(c, r, 200)
				for {
					if clientID%10 == 0 {
						// slow client
						time.Sleep(time.Second / 20)
					} else if rand.Float64() < 0.01 {
						// disconnect+reconnect
						break
					}
					var lg arvados.Log
					err := r.Decode(&lg)
					if ctx.Err() != nil {
						return
					}
					if errors.Is(err, io.EOF) {
						break
					}
					c.Check(err, check.IsNil)
					if i := int(time.Since(t0) / time.Second); i < seconds {
						atomic.AddInt64(&receivedPerSecond[i], 1)
					}
				}
				conn.Close()
			}
		}()
	}
	<-ctx.Done()
	c.Log("done")
	for i, n := range receivedPerSecond {
		c.Logf("t<%d n=%d", i+1, n)
		c.Check(int64(n), check.Not(check.Equals), int64(0))
	}
}

// Generate some events by creating and updating a workflow object,
// and creating a custom log entry (event_type="blip") about the newly
// created workflow.
//
// If uuidChan is not nil, send the new workflow UUID to uuidChan as
// soon as it's known.
//
// If done is not nil, keep generating events until done receives or
// closes.
func (s *v0Suite) emitEvents(c *check.C, uuidChan chan<- string, done <-chan struct{}) {
	s.wg.Add(1)
	defer s.wg.Done()

	ac := arvados.NewClientFromEnv()
	ac.AuthToken = s.token
	wf := &arvados.Workflow{
		Name: "ws_test",
	}
	err := ac.RequestAndDecode(wf, "POST", "arvados/v1/workflows", s.jsonBody("workflow", `{"name":"ws_test"}`), map[string]interface{}{"ensure_unique_name": true})
	c.Assert(err, check.IsNil)
	s.toDelete = append(s.toDelete, "arvados/v1/workflows/"+wf.UUID)
	if uuidChan != nil {
		uuidChan <- wf.UUID
	}
	for i := 0; ; i++ {
		lg := &arvados.Log{}
		err = ac.RequestAndDecode(lg, "POST", "arvados/v1/logs", s.jsonBody("log", map[string]interface{}{
			"object_uuid": wf.UUID,
			"event_type":  "blip",
			"properties": map[string]interface{}{
				"beep": "boop",
			},
		}), nil)
		s.toDelete = append(s.toDelete, "arvados/v1/logs/"+lg.UUID)
		if done != nil {
			select {
			case <-done:
			default:
				if i%50 == 0 {
					time.Sleep(100 * time.Millisecond)
				}
				continue
			}
		}
		break
	}
	if err != nil {
		panic(err)
	}
	err = ac.RequestAndDecode(wf, "PUT", "arvados/v1/workflows/"+wf.UUID, s.jsonBody("workflow", `{"name":"ws_test"}`), nil)
	if err != nil {
		panic(err)
	}
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

func (s *v0Suite) testClient() (*websocket.Conn, *json.Decoder, *json.Encoder, error) {
	srv := s.serviceSuite.srv
	conn, err := websocket.Dial(strings.Replace(srv.URL, "http", "ws", 1)+"/websocket?api_token="+s.token, "", srv.URL)
	if err != nil {
		return nil, nil, nil, err
	}
	w := json.NewEncoder(conn)
	r := json.NewDecoder(conn)
	return conn, r, w, nil
}

func (s *v0Suite) lastLogID(c *check.C) int64 {
	var lastID int64
	c.Assert(testDB().QueryRow(`SELECT MAX(id) FROM logs`).Scan(&lastID), check.IsNil)
	return lastID
}
