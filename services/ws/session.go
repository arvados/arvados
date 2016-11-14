package main

type session interface {
	Receive(map[string]interface{})
	EventMessage(*event) ([]byte, error)
	Filter(*event) bool
	debugLogf(string, ...interface{})
}
