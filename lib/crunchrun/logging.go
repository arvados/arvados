// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
)

// Timestamper is the signature for a function that takes a timestamp and
// return a formated string value.
type Timestamper func(t time.Time) string

// Logging plumbing:
//
// ThrottledLogger.Logger -> ThrottledLogger.Write ->
// ThrottledLogger.buf -> ThrottledLogger.flusher ->
// ArvLogWriter.Write -> CollectionFileWriter.Write | Api.Create
//
// For stdout/stderr ReadWriteLines additionally runs as a goroutine to pull
// data from the stdout/stderr Reader and send to the Logger.

// ThrottledLogger accepts writes, prepends a timestamp to each line of the
// write, and periodically flushes to a downstream writer.  It supports the
// "Logger" and "WriteCloser" interfaces.
type ThrottledLogger struct {
	*log.Logger
	buf *bytes.Buffer
	sync.Mutex
	writer   io.WriteCloser
	flush    chan struct{}
	stopped  chan struct{}
	stopping chan struct{}
	Timestamper
	Immediate    *log.Logger
	pendingFlush bool
}

// RFC3339NanoFixed is a fixed-width version of time.RFC3339Nano.
const RFC3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// RFC3339Timestamp formats t as RFC3339NanoFixed.
func RFC3339Timestamp(t time.Time) string {
	return t.Format(RFC3339NanoFixed)
}

// Write prepends a timestamp to each line of the input data and
// appends to the internal buffer. Each line is also logged to
// tl.Immediate, if tl.Immediate is not nil.
func (tl *ThrottledLogger) Write(p []byte) (n int, err error) {
	tl.Mutex.Lock()
	defer tl.Mutex.Unlock()

	if tl.buf == nil {
		tl.buf = &bytes.Buffer{}
	}

	now := tl.Timestamper(time.Now().UTC())
	sc := bufio.NewScanner(bytes.NewBuffer(p))
	for err == nil && sc.Scan() {
		out := fmt.Sprintf("%s %s\n", now, sc.Bytes())
		if tl.Immediate != nil {
			tl.Immediate.Print(out[:len(out)-1])
		}
		_, err = io.WriteString(tl.buf, out)
	}
	if err == nil {
		err = sc.Err()
		if err == nil {
			n = len(p)
		}
	}

	if int64(tl.buf.Len()) >= crunchLogBytesPerEvent {
		// Non-blocking send.  Try send a flush if it is ready to
		// accept it.  Otherwise do nothing because a flush is already
		// pending.
		select {
		case tl.flush <- struct{}{}:
		default:
		}
	}

	return
}

// Periodically check the current buffer; if not empty, send it on the
// channel to the goWriter goroutine.
func (tl *ThrottledLogger) flusher() {
	ticker := time.NewTicker(time.Duration(crunchLogSecondsBetweenEvents))
	defer ticker.Stop()
	for stopping := false; !stopping; {
		select {
		case <-tl.stopping:
			// flush tl.buf and exit the loop
			stopping = true
		case <-tl.flush:
		case <-ticker.C:
		}

		var ready *bytes.Buffer

		tl.Mutex.Lock()
		ready, tl.buf = tl.buf, &bytes.Buffer{}
		tl.Mutex.Unlock()

		if ready != nil && ready.Len() > 0 {
			tl.writer.Write(ready.Bytes())
		}
	}
	close(tl.stopped)
}

// Close the flusher goroutine and wait for it to complete, then close the
// underlying Writer.
func (tl *ThrottledLogger) Close() error {
	select {
	case <-tl.stopping:
		// already stopped
	default:
		close(tl.stopping)
	}
	<-tl.stopped
	return tl.writer.Close()
}

const (
	// MaxLogLine is the maximum length of stdout/stderr lines before they are split.
	MaxLogLine = 1 << 12
)

// ReadWriteLines reads lines from a reader and writes to a Writer, with long
// line splitting.
func ReadWriteLines(in io.Reader, writer io.Writer, done chan<- bool) {
	reader := bufio.NewReaderSize(in, MaxLogLine)
	var prefix string
	for {
		line, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			writer.Write([]byte(fmt.Sprintln("error reading container log:", err)))
		}
		var suffix string
		if isPrefix {
			suffix = "[...]\n"
		}

		if prefix == "" && suffix == "" {
			writer.Write(line)
		} else {
			writer.Write([]byte(fmt.Sprint(prefix, string(line), suffix)))
		}

		// Set up prefix for following line
		if isPrefix {
			prefix = "[...]"
		} else {
			prefix = ""
		}
	}
	done <- true
}

// NewThrottledLogger creates a new thottled logger that
// (a) prepends timestamps to each line
// (b) batches log messages and only calls the underlying Writer
//  at most once per "crunchLogSecondsBetweenEvents" seconds.
func NewThrottledLogger(writer io.WriteCloser) *ThrottledLogger {
	tl := &ThrottledLogger{}
	tl.flush = make(chan struct{}, 1)
	tl.stopped = make(chan struct{})
	tl.stopping = make(chan struct{})
	tl.writer = writer
	tl.Logger = log.New(tl, "", 0)
	tl.Timestamper = RFC3339Timestamp
	go tl.flusher()
	return tl
}

// Log throttling rate limiting config parameters
var crunchLimitLogBytesPerJob int64 = 67108864
var crunchLogThrottleBytes int64 = 65536
var crunchLogThrottlePeriod time.Duration = time.Second * 60
var crunchLogThrottleLines int64 = 1024
var crunchLogPartialLineThrottlePeriod time.Duration = time.Second * 5
var crunchLogBytesPerEvent int64 = 4096
var crunchLogSecondsBetweenEvents = time.Second
var crunchLogUpdatePeriod = time.Hour / 2
var crunchLogUpdateSize = int64(1 << 25)

// ArvLogWriter is an io.WriteCloser that processes each write by
// writing it through to another io.WriteCloser (typically a
// CollectionFileWriter) and creating an Arvados log entry.
type ArvLogWriter struct {
	ArvClient     IArvadosClient
	UUID          string
	loggingStream string
	writeCloser   io.WriteCloser

	// for rate limiting
	bytesLogged                  int64
	logThrottleResetTime         time.Time
	logThrottleLinesSoFar        int64
	logThrottleBytesSoFar        int64
	logThrottleBytesSkipped      int64
	logThrottleIsOpen            bool
	logThrottlePartialLineNextAt time.Time
	logThrottleFirstPartialLine  bool
	bufToFlush                   bytes.Buffer
	bufFlushedAt                 time.Time
	closing                      bool
}

func (arvlog *ArvLogWriter) Write(p []byte) (int, error) {
	// Write to the next writer in the chain (a file in Keep)
	var err1 error
	if arvlog.writeCloser != nil {
		_, err1 = arvlog.writeCloser.Write(p)
	}

	// write to API after checking rate limit
	now := time.Now()

	if now.After(arvlog.logThrottleResetTime) {
		// It has been more than throttle_period seconds since the last
		// checkpoint; so reset the throttle
		if arvlog.logThrottleBytesSkipped > 0 {
			arvlog.bufToFlush.WriteString(fmt.Sprintf("%s Skipped %d bytes of log\n", RFC3339Timestamp(now.UTC()), arvlog.logThrottleBytesSkipped))
		}

		arvlog.logThrottleResetTime = now.Add(crunchLogThrottlePeriod)
		arvlog.logThrottleBytesSoFar = 0
		arvlog.logThrottleLinesSoFar = 0
		arvlog.logThrottleBytesSkipped = 0
		arvlog.logThrottleIsOpen = true
	}

	lines := bytes.Split(p, []byte("\n"))

	for _, line := range lines {
		// Short circuit the counting code if we're just going to throw
		// away the data anyway.
		if !arvlog.logThrottleIsOpen {
			arvlog.logThrottleBytesSkipped += int64(len(line))
			continue
		} else if len(line) == 0 {
			continue
		}

		// check rateLimit
		logOpen, msg := arvlog.rateLimit(line, now)
		if logOpen {
			arvlog.bufToFlush.WriteString(string(msg) + "\n")
		}
	}

	if (int64(arvlog.bufToFlush.Len()) >= crunchLogBytesPerEvent ||
		(now.Sub(arvlog.bufFlushedAt) >= crunchLogSecondsBetweenEvents) ||
		arvlog.closing) && (arvlog.bufToFlush.Len() > 0) {
		// write to API
		lr := arvadosclient.Dict{"log": arvadosclient.Dict{
			"object_uuid": arvlog.UUID,
			"event_type":  arvlog.loggingStream,
			"properties":  map[string]string{"text": arvlog.bufToFlush.String()}}}
		err2 := arvlog.ArvClient.Create("logs", lr, nil)

		arvlog.bufToFlush = bytes.Buffer{}
		arvlog.bufFlushedAt = now

		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("%s ; %s", err1, err2)
		}
	}

	return len(p), nil
}

// Close the underlying writer
func (arvlog *ArvLogWriter) Close() (err error) {
	arvlog.closing = true
	arvlog.Write([]byte{})
	if arvlog.writeCloser != nil {
		err = arvlog.writeCloser.Close()
		arvlog.writeCloser = nil
	}
	return err
}

var lineRegexp = regexp.MustCompile(`^\S+ (.*)`)

// Test for hard cap on total output and for log throttling. Returns whether
// the log line should go to output or not. Returns message if limit exceeded.
func (arvlog *ArvLogWriter) rateLimit(line []byte, now time.Time) (bool, []byte) {
	message := ""
	lineSize := int64(len(line))

	if arvlog.logThrottleIsOpen {
		matches := lineRegexp.FindStringSubmatch(string(line))

		if len(matches) == 2 && strings.HasPrefix(matches[1], "[...]") && strings.HasSuffix(matches[1], "[...]") {
			// This is a partial line.

			if arvlog.logThrottleFirstPartialLine {
				// Partial should be suppressed.  First time this is happening for this line so provide a message instead.
				arvlog.logThrottleFirstPartialLine = false
				arvlog.logThrottlePartialLineNextAt = now.Add(crunchLogPartialLineThrottlePeriod)
				arvlog.logThrottleBytesSkipped += lineSize
				return true, []byte(fmt.Sprintf("%s Rate-limiting partial segments of long lines to one every %d seconds.",
					RFC3339Timestamp(now.UTC()), crunchLogPartialLineThrottlePeriod/time.Second))
			} else if now.After(arvlog.logThrottlePartialLineNextAt) {
				// The throttle period has passed.  Update timestamp and let it through.
				arvlog.logThrottlePartialLineNextAt = now.Add(crunchLogPartialLineThrottlePeriod)
			} else {
				// Suppress line.
				arvlog.logThrottleBytesSkipped += lineSize
				return false, line
			}
		} else {
			// Not a partial line so reset.
			arvlog.logThrottlePartialLineNextAt = time.Time{}
			arvlog.logThrottleFirstPartialLine = true
		}

		arvlog.bytesLogged += lineSize
		arvlog.logThrottleBytesSoFar += lineSize
		arvlog.logThrottleLinesSoFar++

		if arvlog.bytesLogged > crunchLimitLogBytesPerJob {
			message = fmt.Sprintf("%s Exceeded log limit %d bytes (crunch_limit_log_bytes_per_job). Log will be truncated.",
				RFC3339Timestamp(now.UTC()), crunchLimitLogBytesPerJob)
			arvlog.logThrottleResetTime = now.Add(time.Duration(365 * 24 * time.Hour))
			arvlog.logThrottleIsOpen = false

		} else if arvlog.logThrottleBytesSoFar > crunchLogThrottleBytes {
			remainingTime := arvlog.logThrottleResetTime.Sub(now)
			message = fmt.Sprintf("%s Exceeded rate %d bytes per %d seconds (crunch_log_throttle_bytes). Logging will be silenced for the next %d seconds.",
				RFC3339Timestamp(now.UTC()), crunchLogThrottleBytes, crunchLogThrottlePeriod/time.Second, remainingTime/time.Second)
			arvlog.logThrottleIsOpen = false

		} else if arvlog.logThrottleLinesSoFar > crunchLogThrottleLines {
			remainingTime := arvlog.logThrottleResetTime.Sub(now)
			message = fmt.Sprintf("%s Exceeded rate %d lines per %d seconds (crunch_log_throttle_lines), logging will be silenced for the next %d seconds.",
				RFC3339Timestamp(now.UTC()), crunchLogThrottleLines, crunchLogThrottlePeriod/time.Second, remainingTime/time.Second)
			arvlog.logThrottleIsOpen = false

		}
	}

	if !arvlog.logThrottleIsOpen {
		// Don't log anything if any limit has been exceeded. Just count lossage.
		arvlog.logThrottleBytesSkipped += lineSize
	}

	if message != "" {
		// Yes, write to logs, but use our "rate exceeded" message
		// instead of the log message that exceeded the limit.
		message += " A complete log is still being written to Keep, and will be available when the job finishes."
		return true, []byte(message)
	}
	return arvlog.logThrottleIsOpen, line
}

// load the rate limit discovery config parameters
func loadLogThrottleParams(clnt IArvadosClient) {
	loadDuration := func(dst *time.Duration, key string) {
		if param, err := clnt.Discovery(key); err != nil {
			return
		} else if d, ok := param.(float64); !ok {
			return
		} else {
			*dst = time.Duration(d) * time.Second
		}
	}
	loadInt64 := func(dst *int64, key string) {
		if param, err := clnt.Discovery(key); err != nil {
			return
		} else if val, ok := param.(float64); !ok {
			return
		} else {
			*dst = int64(val)
		}
	}

	loadInt64(&crunchLimitLogBytesPerJob, "crunchLimitLogBytesPerJob")
	loadInt64(&crunchLogThrottleBytes, "crunchLogThrottleBytes")
	loadDuration(&crunchLogThrottlePeriod, "crunchLogThrottlePeriod")
	loadInt64(&crunchLogThrottleLines, "crunchLogThrottleLines")
	loadDuration(&crunchLogPartialLineThrottlePeriod, "crunchLogPartialLineThrottlePeriod")
	loadInt64(&crunchLogBytesPerEvent, "crunchLogBytesPerEvent")
	loadDuration(&crunchLogSecondsBetweenEvents, "crunchLogSecondsBetweenEvents")
	loadInt64(&crunchLogUpdateSize, "crunchLogUpdateSize")
	loadDuration(&crunchLogUpdatePeriod, "crunchLogUpdatePeriod")

}

type filterKeepstoreErrorsOnly struct {
	io.WriteCloser
	buf []byte
}

func (f *filterKeepstoreErrorsOnly) Write(p []byte) (int, error) {
	log.Printf("filterKeepstoreErrorsOnly: write %q", p)
	f.buf = append(f.buf, p...)
	start := 0
	for i := len(f.buf) - len(p); i < len(f.buf); i++ {
		if f.buf[i] == '\n' {
			if f.check(f.buf[start:i]) {
				_, err := f.WriteCloser.Write(f.buf[start : i+1])
				if err != nil {
					return 0, err
				}
			}
			start = i + 1
		}
	}
	if start > 0 {
		copy(f.buf, f.buf[start:])
		f.buf = f.buf[:len(f.buf)-start]
	}
	return len(p), nil
}

func (f *filterKeepstoreErrorsOnly) check(line []byte) bool {
	if len(line) == 0 {
		return false
	}
	if line[0] != '{' {
		return true
	}
	var m map[string]interface{}
	err := json.Unmarshal(line, &m)
	if err != nil {
		return true
	}
	if m["msg"] == "request" {
		return false
	}
	if m["msg"] == "response" {
		if code, _ := m["respStatusCode"].(float64); code >= 200 && code < 300 {
			return false
		}
	}
	return true
}
