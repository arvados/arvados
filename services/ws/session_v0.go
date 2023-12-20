// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

var (
	errQueueFull   = errors.New("client queue full")
	errFrameTooBig = errors.New("frame too big")

	// Send clients only these keys from the
	// log.properties.old_attributes and
	// log.properties.new_attributes hashes.
	sendObjectAttributes = []string{
		"is_trashed",
		"name",
		"owner_uuid",
		"portable_data_hash",
		"requesting_container_uuid",
		"state",
	}

	v0subscribeOK   = []byte(`{"status":200}`)
	v0subscribeFail = []byte(`{"status":400}`)
)

type v0session struct {
	ac            *arvados.Client
	ws            wsConn
	sendq         chan<- interface{}
	db            *sql.DB
	permChecker   permChecker
	subscriptions []v0subscribe
	lastMsgID     uint64
	log           logrus.FieldLogger
	mtx           sync.Mutex
	setupOnce     sync.Once
}

// newSessionV0 returns a v0 session: a partial port of the Rails/puma
// implementation, with just enough functionality to support Workbench
// and arv-mount.
func newSessionV0(ws wsConn, sendq chan<- interface{}, db *sql.DB, pc permChecker, ac *arvados.Client) (session, error) {
	sess := &v0session{
		sendq:       sendq,
		ws:          ws,
		db:          db,
		ac:          ac,
		permChecker: pc,
		log:         ctxlog.FromContext(ws.Request().Context()),
	}

	err := ws.Request().ParseForm()
	if err != nil {
		sess.log.WithError(err).Error("ParseForm failed")
		return nil, err
	}
	token := ws.Request().Form.Get("api_token")
	sess.permChecker.SetToken(token)
	sess.log.WithField("token", token).Debug("set token")

	return sess, nil
}

func (sess *v0session) Receive(buf []byte) error {
	var sub v0subscribe
	if err := json.Unmarshal(buf, &sub); err != nil {
		sess.log.WithError(err).Info("invalid message from client")
	} else if sub.Method == "subscribe" {
		sub.prepare(sess)
		sess.log.WithField("sub", sub).Debug("sub prepared")
		sess.sendq <- v0subscribeOK
		sess.mtx.Lock()
		sess.subscriptions = append(sess.subscriptions, sub)
		sess.mtx.Unlock()
		sub.sendOldEvents(sess)
		return nil
	} else if sub.Method == "unsubscribe" {
		sess.mtx.Lock()
		found := false
		for i, s := range sess.subscriptions {
			if !reflect.DeepEqual(s.Filters, sub.Filters) {
				continue
			}
			copy(sess.subscriptions[i:], sess.subscriptions[i+1:])
			sess.subscriptions = sess.subscriptions[:len(sess.subscriptions)-1]
			found = true
			break
		}
		sess.mtx.Unlock()
		sess.log.WithField("sub", sub).WithField("found", found).Debug("unsubscribe")
		if found {
			sess.sendq <- v0subscribeOK
			return nil
		}
	} else {
		sess.log.WithField("Method", sub.Method).Info("unknown method")
	}
	sess.sendq <- v0subscribeFail
	return nil
}

func (sess *v0session) EventMessage(e *event) ([]byte, error) {
	detail := e.Detail()
	if detail == nil {
		return nil, nil
	}

	var permTarget string
	if detail.EventType == "delete" {
		// It's pointless to check permission by reading
		// ObjectUUID if it has just been deleted, but if the
		// client has permission on the parent project then
		// it's OK to send the event.
		permTarget = detail.ObjectOwnerUUID
	} else {
		permTarget = detail.ObjectUUID
	}
	ok, err := sess.permChecker.Check(sess.ws.Request().Context(), permTarget)
	if err != nil || !ok {
		return nil, err
	}

	kind, _ := sess.ac.KindForUUID(detail.ObjectUUID)
	msg := map[string]interface{}{
		"msgID":             atomic.AddUint64(&sess.lastMsgID, 1),
		"id":                detail.ID,
		"uuid":              detail.UUID,
		"object_uuid":       detail.ObjectUUID,
		"object_owner_uuid": detail.ObjectOwnerUUID,
		"object_kind":       kind,
		"event_type":        detail.EventType,
		"event_at":          detail.EventAt,
	}
	if detail.Properties != nil && detail.Properties["text"] != nil {
		msg["properties"] = detail.Properties
	} else {
		msgProps := map[string]map[string]interface{}{}
		for _, ak := range []string{"old_attributes", "new_attributes"} {
			eventAttrs, ok := detail.Properties[ak].(map[string]interface{})
			if !ok {
				continue
			}
			msgAttrs := map[string]interface{}{}
			for _, k := range sendObjectAttributes {
				if v, ok := eventAttrs[k]; ok {
					msgAttrs[k] = v
				}
			}
			msgProps[ak] = msgAttrs
		}
		msg["properties"] = msgProps
	}
	return json.Marshal(msg)
}

func (sess *v0session) Filter(e *event) bool {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()
	for _, sub := range sess.subscriptions {
		if sub.match(sess, e) {
			return true
		}
	}
	return false
}

func (sub *v0subscribe) sendOldEvents(sess *v0session) {
	if sub.LastLogID == 0 {
		return
	}
	sess.log.WithField("LastLogID", sub.LastLogID).Debug("sendOldEvents")
	// Here we do a "select id" query and queue an event for every
	// log since the given ID, then use (*event)Detail() to
	// retrieve the whole row and decide whether to send it. This
	// approach is very inefficient if the subscriber asks for
	// last_log_id==1, even if the filters end up matching very
	// few events.
	//
	// To mitigate this, filter on "created > 10 minutes ago" when
	// retrieving the list of old event IDs to consider.
	rows, err := sess.db.Query(
		`SELECT id FROM logs WHERE id > $1 AND created_at > $2 ORDER BY id`,
		sub.LastLogID,
		time.Now().UTC().Add(-10*time.Minute).Format(time.RFC3339Nano))
	if err != nil {
		sess.log.WithError(err).Error("sendOldEvents db.Query failed")
		return
	}

	var ids []int64
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
		if err != nil {
			sess.log.WithError(err).Error("sendOldEvents row Scan failed")
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		sess.log.WithError(err).Error("sendOldEvents db.Query failed")
	}
	rows.Close()

	for _, id := range ids {
		for len(sess.sendq)*2 > cap(sess.sendq) {
			// Ugly... but if we fill up the whole client
			// queue with a backlog of old events, a
			// single new event will overflow it and
			// terminate the connection, and then the
			// client will probably reconnect and do the
			// same thing all over again.
			time.Sleep(100 * time.Millisecond)
			if sess.ws.Request().Context().Err() != nil {
				// Session terminated while we were sleeping
				return
			}
		}
		now := time.Now()
		e := &event{
			LogID:    id,
			Received: now,
			Ready:    now,
			db:       sess.db,
		}
		if sub.match(sess, e) {
			select {
			case sess.sendq <- e:
			case <-sess.ws.Request().Context().Done():
				return
			}
		}
	}
}

type v0subscribe struct {
	Method    string
	Filters   []v0filter
	LastLogID int64 `json:"last_log_id"`

	funcs []func(*event) bool
}

type v0filter [3]interface{}

func (sub *v0subscribe) match(sess *v0session, e *event) bool {
	log := sess.log.WithField("LogID", e.LogID)
	detail := e.Detail()
	if detail == nil {
		log.Error("match failed, no detail")
		return false
	}
	log = log.WithField("funcs", len(sub.funcs))
	for i, f := range sub.funcs {
		if !f(e) {
			log.WithField("func", i).Debug("match failed")
			return false
		}
	}
	log.Debug("match passed")
	return true
}

func (sub *v0subscribe) prepare(sess *v0session) {
	for _, f := range sub.Filters {
		if len(f) != 3 {
			continue
		}
		if col, ok := f[0].(string); ok && col == "event_type" {
			op, ok := f[1].(string)
			if !ok || op != "in" {
				continue
			}
			arr, ok := f[2].([]interface{})
			if !ok {
				continue
			}
			var strs []string
			for _, s := range arr {
				if s, ok := s.(string); ok {
					strs = append(strs, s)
				}
			}
			sub.funcs = append(sub.funcs, func(e *event) bool {
				for _, s := range strs {
					if s == e.Detail().EventType {
						return true
					}
				}
				return false
			})
		} else if ok && col == "created_at" {
			op, ok := f[1].(string)
			if !ok {
				continue
			}
			tstr, ok := f[2].(string)
			if !ok {
				continue
			}
			t, err := time.Parse(time.RFC3339Nano, tstr)
			if err != nil {
				sess.log.WithField("data", tstr).WithError(err).Info("time.Parse failed")
				continue
			}
			var fn func(*event) bool
			switch op {
			case ">=":
				fn = func(e *event) bool {
					return !e.Detail().CreatedAt.Before(t)
				}
			case "<=":
				fn = func(e *event) bool {
					return !e.Detail().CreatedAt.After(t)
				}
			case ">":
				fn = func(e *event) bool {
					return e.Detail().CreatedAt.After(t)
				}
			case "<":
				fn = func(e *event) bool {
					return e.Detail().CreatedAt.Before(t)
				}
			case "=":
				fn = func(e *event) bool {
					return e.Detail().CreatedAt.Equal(t)
				}
			default:
				sess.log.WithField("operator", op).Info("bogus operator")
				continue
			}
			sub.funcs = append(sub.funcs, fn)
		}
	}
}
