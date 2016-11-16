package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var (
	errQueueFull   = errors.New("client queue full")
	errFrameTooBig = errors.New("frame too big")

	sendObjectAttributes = []string{"state", "name"}

	v0subscribeOK   = []byte(`{"status":200}`)
	v0subscribeFail = []byte(`{"status":400}`)
)

type v0session struct {
	ws            wsConn
	db            *sql.DB
	permChecker   permChecker
	subscriptions []v0subscribe
	mtx           sync.Mutex
	setupOnce     sync.Once
}

func NewSessionV0(ws wsConn, ac arvados.Client, db *sql.DB) (session, error) {
	sess := &v0session{
		ws:          ws,
		db:          db,
		permChecker: NewPermChecker(ac),
	}

	err := ws.Request().ParseForm()
	if err != nil {
		log.Printf("%s ParseForm: %s", ws.Request().RemoteAddr, err)
		return nil, err
	}
	token := ws.Request().Form.Get("api_token")
	sess.permChecker.SetToken(token)
	sess.debugLogf("token = %+q", token)

	return sess, nil
}

func (sess *v0session) debugLogf(s string, args ...interface{}) {
	args = append([]interface{}{sess.ws.Request().RemoteAddr}, args...)
	debugLogf("%s "+s, args...)
}

func (sess *v0session) Receive(msg map[string]interface{}, buf []byte) [][]byte {
	sess.debugLogf("received message: %+v", msg)
	var sub v0subscribe
	if err := json.Unmarshal(buf, &sub); err != nil {
		sess.debugLogf("ignored unrecognized request: %s", err)
		return nil
	}
	if sub.Method == "subscribe" {
		sub.prepare()
		sess.debugLogf("subscription: %v", sub)
		sess.mtx.Lock()
		sess.subscriptions = append(sess.subscriptions, sub)
		sess.mtx.Unlock()

		return append([][]byte{v0subscribeOK}, sub.getOldEvents(sess)...)
	}
	return [][]byte{v0subscribeFail}
}

func (sess *v0session) EventMessage(e *event) ([]byte, error) {
	detail := e.Detail()
	if detail == nil {
		return nil, nil
	}

	ok, err := sess.permChecker.Check(detail.ObjectUUID)
	if err != nil || !ok {
		return nil, err
	}

	msg := map[string]interface{}{
		"msgID":             e.Serial,
		"id":                detail.ID,
		"uuid":              detail.UUID,
		"object_uuid":       detail.ObjectUUID,
		"object_owner_uuid": detail.ObjectOwnerUUID,
		"event_type":        detail.EventType,
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
		if sub.match(e) {
			return true
		}
	}
	return false
}

func (sub *v0subscribe) getOldEvents(sess *v0session) (msgs [][]byte) {
	if sub.LastLogID == 0 {
		return
	}
	debugLogf("getOldEvents(%d)", sub.LastLogID)
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
		errorLogf("db.Query: %s", err)
		return
	}
	for rows.Next() {
		var id uint64
		err := rows.Scan(&id)
		if err != nil {
			errorLogf("Scan: %s", err)
			continue
		}
		e := &event{
			LogID:    id,
			Received: time.Now(),
			db:       sess.db,
		}
		if !sub.match(e) {
			debugLogf("skip old event %+v", e)
			continue
		}
		msg, err := sess.EventMessage(e)
		if err != nil {
			debugLogf("event marshal: %s", err)
			continue
		}
		debugLogf("old event: %s", string(msg))
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		errorLogf("db.Query: %s", err)
	}
	return
}

type v0subscribe struct {
	Method    string
	Filters   []v0filter
	LastLogID int64 `json:"last_log_id"`

	funcs []func(*event) bool
}

type v0filter [3]interface{}

func (sub *v0subscribe) match(e *event) bool {
	detail := e.Detail()
	if detail == nil {
		debugLogf("match(%d): failed on no detail", e.LogID)
		return false
	}
	for i, f := range sub.funcs {
		if !f(e) {
			debugLogf("match(%d): failed on func %d", e.LogID, i)
			return false
		}
	}
	debugLogf("match(%d): passed %d funcs", e.LogID, len(sub.funcs))
	return true
}

func (sub *v0subscribe) prepare() {
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
				debugLogf("event_type func: %v in %v", e.Detail().EventType, strs)
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
				debugLogf("time.Parse(%q): %s", tstr, err)
				continue
			}
			switch op {
			case ">=":
				sub.funcs = append(sub.funcs, func(e *event) bool {
					debugLogf("created_at func: %v >= %v", e.Detail().CreatedAt, t)
					return !e.Detail().CreatedAt.Before(t)
				})
			case "<=":
				sub.funcs = append(sub.funcs, func(e *event) bool {
					debugLogf("created_at func: %v <= %v", e.Detail().CreatedAt, t)
					return !e.Detail().CreatedAt.After(t)
				})
			case ">":
				sub.funcs = append(sub.funcs, func(e *event) bool {
					debugLogf("created_at func: %v > %v", e.Detail().CreatedAt, t)
					return e.Detail().CreatedAt.After(t)
				})
			case "<":
				sub.funcs = append(sub.funcs, func(e *event) bool {
					debugLogf("created_at func: %v < %v", e.Detail().CreatedAt, t)
					return e.Detail().CreatedAt.Before(t)
				})
			case "=":
				sub.funcs = append(sub.funcs, func(e *event) bool {
					debugLogf("created_at func: %v = %v", e.Detail().CreatedAt, t)
					return e.Detail().CreatedAt.Equal(t)
				})
			}
		}
	}
}
