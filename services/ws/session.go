package main

type session interface {
	// Receive processes a message received from the client. If
	// the returned response is non-nil, it will be queued and
	// sent the client.
	Receive(map[string]interface{}, []byte) []byte

	// Filter returns true if the event should be queued for
	// sending to the client. It should return as fast as
	// possible, and must not block.
	Filter(*event) bool

	// EventMessage encodes the given event (from the front of the
	// queue) into a form suitable to send to the client. If a
	// non-nil error is returned, the connection is terminated. If
	// the returned buffer is empty, nothing is sent to the client
	// and the event is not counted in statistics.
	//
	// Unlike Filter, EventMessage can block without affecting
	// other connections. If EventMessage is slow, additional
	// incoming events will be queued. If the event queue fills
	// up, the connection will be dropped.
	EventMessage(*event) ([]byte, error)

	debugLogf(string, ...interface{})
}
