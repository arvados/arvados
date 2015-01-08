// Periodically writes a log to the Arvados SDK.
//
// This package is useful for maintaining a log object that is built
// up over time. Every time the object is modified, it will be written
// to the log. Writes will be throttled to no more than one every
// WriteFrequencySeconds
//
// This package is safe for concurrent use.
//
// Usage:
// arvLogger := logger.NewLogger(params)
// logData := arvLogger.Acquire()  // This will block if others are using the logger
// // Modify the logObject however you want here ..
// logData = arvLogger.Release()  // This triggers the actual write, and replaces logObject with a nil pointer so you don't try to modify it when you're no longer holding the lock

package logger

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"sync"
	"time"
)

const (
	eventTypeLabel string = "event-type"
	propertiesLabel string = "properties"
)

type LoggerParams struct {
	Client arvadosclient.ArvadosClient  // The client we use to write log entries
	EventType string  // The event type to assign to the log entry.
	MinimumWriteInterval time.Duration  // Wait at least this long between log writes
}

type Logger struct {
	data map[string]interface{}
	lock sync.Locker
	params LoggerParams
}

func NewLogger(params LoggerParams) *Logger {
	l := &Logger{data: make(map[string]interface{}),
		lock: &sync.Mutex{},
		// TODO(misha): Consider copying the params so they're not
		// modified after creation.
		params: params}
	l.data[propertiesLabel] = make(map[string]interface{})
	return l
}

func (l *Logger) Acquire() map[string]interface{} {
	l.lock.Lock()
	return l.data[propertiesLabel].(map[string]interface{})
}

func (l *Logger) Release() map[string]interface{} {
	// TODO(misha): Add a check (and storage) to make sure we respect MinimumWriteInterval
	l.write()
	l.lock.Unlock()
	return nil
}

func (l *Logger) write() {
	// Update the event type in case it was modified or is missing.
	// l.data[eventTypeLabel] = l.params.EventType
	// m := make(map[string]interface{})
	// m["body"] = l.data
	// //err := l.params.Client.Create("logs", l.data, nil)
	// //err := l.params.Client.Create("logs", m, nil)
	// var results map[string]interface{}
	err := l.params.Client.Create("logs",
		arvadosclient.Dict{"log": arvadosclient.Dict{
			eventTypeLabel: l.params.EventType,
			propertiesLabel: l.data}}, nil)
	if err != nil {
		log.Fatalf("Received error writing log: %v", err)
	}
}
