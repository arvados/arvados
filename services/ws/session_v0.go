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
	mtx         sync.Mutex
	setupOnce   sync.Once
}

func NewSessionV0(ws wsConn, ac arvados.Client) (session, error) {
	sess := &sessionV0{
		ws:          ws,
		permChecker: NewPermChecker(ac),
		subscribed:  make(map[string]bool),
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

func (sess *sessionV0) Receive(msg map[string]interface{}) {
	sess.debugLogf("received message: %+v", msg)
	sess.debugLogf("subscribing to *")
	sess.subscribed["*"] = true
}

func (sess *sessionV0) EventMessage(e *event) ([]byte, error) {
	detail := e.Detail()
	if detail == nil {
		return nil, nil
	}

	ok, err := sess.permChecker.Check(detail.UUID)
	if err != nil || !ok {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"msgID":             e.Serial,
		"id":                detail.ID,
		"uuid":              detail.UUID,
		"object_uuid":       detail.ObjectUUID,
		"object_owner_uuid": detail.ObjectOwnerUUID,
		"event_type":        detail.EventType,
	})
}

func (sess *sessionV0) Filter(e *event) bool {
	detail := e.Detail()
	sess.mtx.Lock()
	defer sess.mtx.Unlock()
	switch {
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
