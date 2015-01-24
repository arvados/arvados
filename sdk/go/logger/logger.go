// Logger periodically writes a log to the Arvados SDK.
//
// This package is useful for maintaining a log object that is updated
// over time. Every time the object is updated, it will be written to
// the log. Writes will be throttled to no more than one every
// WriteFrequencySeconds
//
// This package is safe for concurrent use as long as:
// The maps passed to a LogMutator are not accessed outside of the
// LogMutator
//
// Usage:
// arvLogger := logger.NewLogger(params)
// arvLogger.Update(func(properties map[string]interface{},
// 	entry map[string]interface{}) {
//   // Modifiy properties and entry however you want
//   // properties is a shortcut for entry["properties"].(map[string]interface{})
//   // properties can take any values you want to give it,
//   // entry will only take the fields listed at http://doc.arvados.org/api/schema/Log.html
// })
package logger

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"sync"
	"time"
)

type LoggerParams struct {
	Client               arvadosclient.ArvadosClient // The client we use to write log entries
	EventType            string                      // The event type to assign to the log entry.
	MinimumWriteInterval time.Duration               // Wait at least this long between log writes
}

// A LogMutator is a function which modifies the log entry.
// It takes two maps as arguments, properties is the first and entry
// is the second
// properties is a shortcut for entry["properties"].(map[string]interface{})
// properties can take any values you want to give it.
// entry will only take the fields listed at http://doc.arvados.org/api/schema/Log.html
// properties and entry are only safe to access inside the LogMutator,
// they should not be stored anywhere, otherwise you'll risk
// concurrent access.
type LogMutator func(map[string]interface{}, map[string]interface{})

// A Logger is used to build up a log entry over time and write every
// version of it.
type Logger struct {
	// The data we write
	data       map[string]interface{} // The entire map that we give to the api
	entry      map[string]interface{} // Convenience shortcut into data
	properties map[string]interface{} // Convenience shortcut into data

	lock   sync.Locker  // Synchronizes access to this struct
	params LoggerParams // Parameters we were given

	// Variables used to determine when and if we write to the log.
	nextWriteAllowed time.Time // The next time we can write, respecting MinimumWriteInterval
	modified         bool      // Has this data been modified since the last write?
	writeScheduled   bool      // Is a write been scheduled for the future?

	writeHooks []LogMutator // Mutators we call before each write.
}

// Create a new logger based on the specified parameters.
func NewLogger(params LoggerParams) *Logger {
	// sanity check parameters
	if &params.Client == nil {
		log.Fatal("Nil arvados client in LoggerParams passed in to NewLogger()")
	}
	if params.EventType == "" {
		log.Fatal("Empty event type in LoggerParams passed in to NewLogger()")
	}

	l := &Logger{data: make(map[string]interface{}),
		lock:   &sync.Mutex{},
		params: params}
	l.entry = make(map[string]interface{})
	l.data["log"] = l.entry
	l.properties = make(map[string]interface{})
	l.entry["properties"] = l.properties
	return l
}

// Updates the log data and then writes it to the api server. If the
// log has been recently written then the write will be postponed to
// respect MinimumWriteInterval and this function will return before
// the write occurs.
func (l *Logger) Update(mutator LogMutator) {
	l.lock.Lock()

	mutator(l.properties, l.entry)
	l.modified = true // We assume the mutator modified the log, even though we don't know for sure.

	l.considerWriting()

	l.lock.Unlock()
}

// Similar to Update(), but forces a write without respecting the
// MinimumWriteInterval. This is useful if you know that you're about
// to quit (e.g. if you discovered a fatal error, or you're finished),
// since go will not wait for timers (including the pending write
// timer) to go off before exiting.
func (l *Logger) ForceUpdate(mutator LogMutator) {
	l.lock.Lock()

	mutator(l.properties, l.entry)
	l.modified = true // We assume the mutator modified the log, even though we don't know for sure.

	l.write()
	l.lock.Unlock()
}

// Adds a hook which will be called every time this logger writes an entry.
func (l *Logger) AddWriteHook(hook LogMutator) {
	l.lock.Lock()
	l.writeHooks = append(l.writeHooks, hook)
	// TODO(misha): Consider setting modified and attempting a write.
	l.lock.Unlock()
}

// This function is called on a timer when we have something to write,
// but need to schedule the write for the future to respect
// MinimumWriteInterval.
func (l *Logger) acquireLockConsiderWriting() {
	l.lock.Lock()

	// We are the scheduled write, so there are no longer future writes
	// scheduled.
	l.writeScheduled = false

	l.considerWriting()

	l.lock.Unlock()
}

// The above methods each acquire the lock and release it.
// =======================================================
// The below methods all assume we're holding a lock.

// Check whether we have anything to write. If we do, then either
// write it now or later, based on what we're allowed.
func (l *Logger) considerWriting() {
	if !l.modified {
		// Nothing to write
	} else if l.writeAllowedNow() {
		l.write()
	} else if l.writeScheduled {
		// A future write is already scheduled, we don't need to do anything.
	} else {
		writeAfter := l.nextWriteAllowed.Sub(time.Now())
		time.AfterFunc(writeAfter, l.acquireLockConsiderWriting)
		l.writeScheduled = true
	}
}

// Whether writing now would respect MinimumWriteInterval
func (l *Logger) writeAllowedNow() bool {
	return l.nextWriteAllowed.Before(time.Now())
}

// Actually writes the log entry.
func (l *Logger) write() {

	// Run all our hooks
	for _, hook := range l.writeHooks {
		hook(l.properties, l.entry)
	}

	// Update the event type in case it was modified or is missing.
	l.entry["event_type"] = l.params.EventType

	// Write the log entry.
	err := l.params.Client.Create("logs", l.data, nil)
	if err != nil {
		log.Printf("Attempted to log: %v", l.data)
		log.Fatalf("Received error writing log: %v", err)
	}

	// Update stats.
	l.nextWriteAllowed = time.Now().Add(l.params.MinimumWriteInterval)
	l.modified = false
}
