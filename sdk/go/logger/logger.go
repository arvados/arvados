// Logger periodically writes a log to the Arvados SDK.
//
// This package is useful for maintaining a log object that is updated
// over time. This log object will be periodically written to the log,
// as specified by WriteInterval in the Params.
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
	"time"
)

const (
	startSuffix   = "-start"
	partialSuffix = "-partial"
	finalSuffix   = "-final"
)

type LoggerParams struct {
	Client          arvadosclient.ArvadosClient // The client we use to write log entries
	EventTypePrefix string                      // The prefix we use for the event type in the log entry
	WriteInterval   time.Duration               // Wait at least this long between log writes
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

	params LoggerParams // Parameters we were given

	// Variables to coordinate updating and writing.
	modified    bool            // Has this data been modified since the last write?
	workToDo    chan LogMutator // Work to do in the worker thread.
	writeTicker *time.Ticker    // On each tick we write the log data to arvados, if it has been modified.
	hasWritten  bool            // Whether we've written at all yet.

	writeHooks []LogMutator // Mutators we call before each write.
}

// Create a new logger based on the specified parameters.
func NewLogger(params LoggerParams) *Logger {
	// sanity check parameters
	if &params.Client == nil {
		log.Fatal("Nil arvados client in LoggerParams passed in to NewLogger()")
	}
	if params.EventTypePrefix == "" {
		log.Fatal("Empty event type prefix in LoggerParams passed in to NewLogger()")
	}

	l := &Logger{data: make(map[string]interface{}),
		params: params}
	l.entry = make(map[string]interface{})
	l.data["log"] = l.entry
	l.properties = make(map[string]interface{})
	l.entry["properties"] = l.properties

	l.workToDo = make(chan LogMutator, 10)
	l.writeTicker = time.NewTicker(params.WriteInterval)

	// Start the worker goroutine.
	go l.work()

	return l
}

// Exported functions will be called from other goroutines, therefore
// all they are allowed to do is enqueue work to be done in the worker
// goroutine.

// Enqueues an update. This will happen in another goroutine after
// this method returns.
func (l *Logger) Update(mutator LogMutator) {
	l.workToDo <- mutator
}

// Similar to Update(), but writes the log entry as soon as possible
// (ignoring MinimumWriteInterval) and blocks until the entry has been
// written. This is useful if you know that you're about to quit
// (e.g. if you discovered a fatal error, or you're finished), since
// go will not wait for timers (including the pending write timer) to
// go off before exiting.
func (l *Logger) FinalUpdate(mutator LogMutator) {
	// Block on this channel until everything finishes
	done := make(chan bool)

	// TODO(misha): Consider not accepting any future updates somehow,
	// since they won't get written if they come in after this.

	// Stop the periodic write ticker. We'll perform the final write
	// before returning from this function.
	l.workToDo <- func(p map[string]interface{}, e map[string]interface{}) {
		l.writeTicker.Stop()
	}

	// Apply the final update
	l.workToDo <- mutator

	// Perform the final write and signal that we can return.
	l.workToDo <- func(p map[string]interface{}, e map[string]interface{}) {
		l.write(true)
		done <- true
	}

	// Wait until we've performed the write.
	<-done
}

// Adds a hook which will be called every time this logger writes an entry.
func (l *Logger) AddWriteHook(hook LogMutator) {
	// We do the work in a LogMutator so that it happens in the worker
	// goroutine.
	l.workToDo <- func(p map[string]interface{}, e map[string]interface{}) {
		l.writeHooks = append(l.writeHooks, hook)
	}
}

// The worker loop
func (l *Logger) work() {
	for {
		select {
		case <-l.writeTicker.C:
			if l.modified {
				l.write(false)
				l.modified = false
			}
		case mutator := <-l.workToDo:
			mutator(l.properties, l.entry)
			l.modified = true
		}
	}
}

// Actually writes the log entry.
func (l *Logger) write(isFinal bool) {

	// Run all our hooks
	for _, hook := range l.writeHooks {
		hook(l.properties, l.entry)
	}

	// Update the event type.
	if isFinal {
		l.entry["event_type"] = l.params.EventTypePrefix + finalSuffix
	} else if l.hasWritten {
		l.entry["event_type"] = l.params.EventTypePrefix + partialSuffix
	} else {
		l.entry["event_type"] = l.params.EventTypePrefix + startSuffix
	}
	l.hasWritten = true

	// Write the log entry.
	// This is a network write and will take a while, which is bad
	// because we're blocking all the other work on this goroutine.
	//
	// TODO(misha): Consider rewriting this so that we can encode l.data
	// into a string, and then perform the actual write in another
	// routine. This will be tricky and will require support in the
	// client.
	err := l.params.Client.Create("logs", l.data, nil)
	if err != nil {
		log.Printf("Attempted to log: %v", l.data)
		log.Fatalf("Received error writing log: %v", err)
	}
}
