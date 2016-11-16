package main

import (
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
	permChecker   permChecker
	subscriptions []v0subscribe
	mtx           sync.Mutex
	setupOnce     sync.Once
}

func NewSessionV0(ws wsConn, ac arvados.Client) (session, error) {
	sess := &v0session{
		ws:          ws,
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
		return [][]byte{v0subscribeOK}
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

type v0subscribe struct {
	Method  string
	Filters []v0filter
	funcs   []func(*event) bool
}

type v0filter [3]interface{}

func (sub *v0subscribe) match(e *event) bool {
	detail := e.Detail()
	if detail == nil {
		return false
	}
	debugLogf("sub.match: len(funcs)==%d", len(sub.funcs))
	for i, f := range sub.funcs {
		if !f(e) {
			debugLogf("sub.match: failed on func %d", i)
			return false
		}
	}
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
