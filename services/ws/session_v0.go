package main

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var (
	errQueueFull   = errors.New("client queue full")
	errFrameTooBig = errors.New("frame too big")
)

type sessionV0 struct {
	ws          wsConn
	permChecker permChecker
	subscribed  map[string]bool
	eventTypes  map[string]bool
	mtx         sync.Mutex
	setupOnce   sync.Once
}

type v0subscribe struct {
	Method  string
	Filters []v0filter
}

type v0filter []interface{}

func NewSessionV0(ws wsConn, ac arvados.Client) (session, error) {
	sess := &sessionV0{
		ws:          ws,
		permChecker: NewPermChecker(ac),
		subscribed:  make(map[string]bool),
		eventTypes:  make(map[string]bool),
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

func (sess *sessionV0) debugLogf(s string, args ...interface{}) {
	args = append([]interface{}{sess.ws.Request().RemoteAddr}, args...)
	debugLogf("%s "+s, args...)
}

// If every client subscription message includes filters consisting
// only of [["event_type","in",...]] then send only the requested
// event types. Otherwise, clear sess.eventTypes and send all event
// types from now on.
func (sess *sessionV0) checkFilters(filters []v0filter) {
	if sess.eventTypes == nil {
		// Already received a subscription request without
		// event_type filters.
		return
	}
	eventTypes := sess.eventTypes
	sess.eventTypes = nil
	if len(filters) == 0 {
		return
	}
	useFilters := false
	for _, f := range filters {
		col, ok := f[0].(string)
		if !ok || col != "event_type" {
			continue
		}
		op, ok := f[1].(string)
		if !ok || op != "in" {
			return
		}
		arr, ok := f[2].([]interface{})
		if !ok {
			return
		}
		useFilters = true
		for _, s := range arr {
			if s, ok := s.(string); ok {
				eventTypes[s] = true
			} else {
				return
			}
		}
	}
	if useFilters {
		sess.debugLogf("eventTypes %+v", eventTypes)
		sess.eventTypes = eventTypes
	}
}

func (sess *sessionV0) Receive(msg map[string]interface{}, buf []byte) {
	sess.debugLogf("received message: %+v", msg)
	var sub v0subscribe
	if err := json.Unmarshal(buf, &sub); err != nil {
		sess.debugLogf("ignored unrecognized request: %s", err)
		return
	}
	if sub.Method == "subscribe" {
		sess.debugLogf("subscribing to *")
		sess.mtx.Lock()
		sess.checkFilters(sub.Filters)
		sess.subscribed["*"] = true
		sess.mtx.Unlock()
	}
}

func (sess *sessionV0) EventMessage(e *event) ([]byte, error) {
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
	}
	return json.Marshal(msg)
}

func (sess *sessionV0) Filter(e *event) bool {
	detail := e.Detail()
	sess.mtx.Lock()
	defer sess.mtx.Unlock()
	switch {
	case sess.eventTypes != nil && detail == nil:
		return false
	case sess.eventTypes != nil && !sess.eventTypes[detail.EventType]:
		return false
	case sess.subscribed["*"]:
		return true
	case detail == nil:
		return false
	case sess.subscribed[detail.ObjectUUID]:
		return true
	case sess.subscribed[detail.ObjectOwnerUUID]:
		return true
	default:
		return false
	}
}
