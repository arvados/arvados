// Logger periodically writes a log to the Arvados SDK.
//
// This package is useful for maintaining a log object that is built
// up over time. Every time the object is modified, it will be written
// to the log. Writes will be throttled to no more than one every
// WriteFrequencySeconds
//
// This package is safe for concurrent use as long as:
// 1. The maps returned by Edit() are only edited in the same routine
//    that called Edit()
// 2. Those maps not edited after calling Record()
// An easy way to assure this is true is to place the call to Edit()
// within a short block as shown below in the Usage Example:
//
// Usage:
// arvLogger := logger.NewLogger(params)
// {
//   properties, entry := arvLogger.Edit()  // This will block if others are using the logger
//   // Modifiy properties and entry however you want
//   // properties is a shortcut for entry["properties"].(map[string]interface{})
//   // properties can take any values you want to give it,
//   // entry will only take the fields listed at http://doc.arvados.org/api/schema/Log.html
// }
// arvLogger.Record()  // This triggers the actual log write
package logger

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"sync"
	"time"
)

type LoggerParams struct {
	Client arvadosclient.ArvadosClient  // The client we use to write log entries
	EventType string  // The event type to assign to the log entry.
	MinimumWriteInterval time.Duration  // Wait at least this long between log writes
}

// A Logger is used to build up a log entry over time and write every
// version of it.
type Logger struct {
	// The Data we write
	data        map[string]interface{}  // The entire map that we give to the api
	entry       map[string]interface{}  // Convenience shortcut into data
	properties  map[string]interface{}  // Convenience shortcut into data

	lock        sync.Locker   // Synchronizes editing and writing
	params      LoggerParams  // Parameters we were given

	lastWrite   time.Time  // The last time we wrote a log entry
	modified    bool       // Has this data been modified since the last write

	writeHooks  []func(map[string]interface{},map[string]interface{})
}

// Create a new logger based on the specified parameters.
func NewLogger(params LoggerParams) *Logger {
	// TODO(misha): Add some params checking here.
	l := &Logger{data: make(map[string]interface{}),
		lock: &sync.Mutex{},
		params: params}
	l.entry = make(map[string]interface{})
	l.data["log"] = l.entry
	l.properties = make(map[string]interface{})
	l.entry["properties"] = l.properties
	return l
}

// Get access to the maps you can edit. This will hold a lock until
// you call Record. Do not edit the maps in any other goroutines or
// after calling Record.
// You don't need to edit both maps, 
// properties can take any values you want to give it,
// entry will only take the fields listed at http://doc.arvados.org/api/schema/Log.html
// properties is a shortcut for entry["properties"].(map[string]interface{})
func (l *Logger) Edit() (properties map[string]interface{}, entry map[string]interface{}) {
	l.lock.Lock()
	l.modified = true  // We don't actually know the caller will modifiy the data, but we assume they will.

	return l.properties, l.entry
}

// Adds a hook which will be called every time this logger writes an entry.
// The hook takes properties and entry as arguments, in that order.
// This is useful for stuff like memory profiling.
// This must be called between Edit() and Record() (e.g. while holding the lock)
func (l *Logger) AddWriteHook(hook func(map[string]interface{},
	map[string]interface{})) {
	l.writeHooks = append(l.writeHooks, hook)
}

// Write the log entry you've built up so far. Do not edit the maps
// returned by Edit() after calling this method.
// If you have already written within MinimumWriteInterval, then this
// will schedule a future write instead.
// In either case, the lock will be released before Record() returns.
func (l *Logger) Record() {
	if l.writeAllowedNow() {
		// We haven't written in the allowed interval yet, try to write.
		l.write()
	} else {
		nextTimeToWrite := l.lastWrite.Add(l.params.MinimumWriteInterval)
		writeAfter := nextTimeToWrite.Sub(time.Now())
		time.AfterFunc(writeAfter, l.acquireLockConsiderWriting)
	}
	l.lock.Unlock()
}

// Similar to Record, but forces a write without respecting the
// MinimumWriteInterval. This is useful if you know that you're about
// to quit (e.g. if you discovered a fatal error).
func (l *Logger) ForceRecord() {
	l.write()
	l.lock.Unlock()
}

// Whether enough time has elapsed since the last write.
func (l *Logger) writeAllowedNow() bool {
	return l.lastWrite.Add(l.params.MinimumWriteInterval).Before(time.Now())
}


// Actually writes the log entry. This method assumes we're holding the lock.
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
	l.lastWrite = time.Now()
	l.modified = false
}


func (l *Logger) acquireLockConsiderWriting() {
	l.lock.Lock()
	if l.modified && l.writeAllowedNow() {
		// We have something new to write and we're allowed to write.
		l.write()
	}
	l.lock.Unlock()
}
