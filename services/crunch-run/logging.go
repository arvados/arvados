package main

import (
	"bufio"
	"bytes"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"io"
	"log"
	"sync"
	"time"
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
	writer      io.WriteCloser
	stop        bool
	flusherDone chan bool
	Timestamper
	Immediate *log.Logger
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
	return
}

// Periodically check the current buffer; if not empty, send it on the
// channel to the goWriter goroutine.
func (tl *ThrottledLogger) flusher() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		// We use a separate "stopping" var here to ensure we flush
		// tl.buf after tl.stop becomes true.
		stopping := tl.stop

		var ready *bytes.Buffer

		tl.Mutex.Lock()
		ready, tl.buf = tl.buf, nil
		tl.Mutex.Unlock()

		if ready != nil && ready.Len() > 0 {
			tl.writer.Write(ready.Bytes())
		}

		if stopping {
			break
		}
	}
	close(tl.flusherDone)
}

// Close the flusher goroutine and wait for it to complete, then close the
// underlying Writer.
func (tl *ThrottledLogger) Close() error {
	tl.stop = true
	<-tl.flusherDone
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
// (b) batches log messages and only calls the underlying Writer at most once
// per second.
func NewThrottledLogger(writer io.WriteCloser) *ThrottledLogger {
	tl := &ThrottledLogger{}
	tl.flusherDone = make(chan bool)
	tl.writer = writer
	tl.Logger = log.New(tl, "", 0)
	tl.Timestamper = RFC3339Timestamp
	go tl.flusher()
	return tl
}

// ArvLogWriter is an io.WriteCloser that processes each write by
// writing it through to another io.WriteCloser (typically a
// CollectionFileWriter) and creating an Arvados log entry.
type ArvLogWriter struct {
	ArvClient     IArvadosClient
	UUID          string
	loggingStream string
	writeCloser   io.WriteCloser
}

func (arvlog *ArvLogWriter) Write(p []byte) (n int, err error) {
	// Write to the next writer in the chain (a file in Keep)
	var err1 error
	if arvlog.writeCloser != nil {
		_, err1 = arvlog.writeCloser.Write(p)
	}

	// write to API
	lr := arvadosclient.Dict{"log": arvadosclient.Dict{
		"object_uuid": arvlog.UUID,
		"event_type":  arvlog.loggingStream,
		"properties":  map[string]string{"text": string(p)}}}
	err2 := arvlog.ArvClient.Create("logs", lr, nil)

	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("%s ; %s", err1, err2)
	}
	return len(p), nil
}

// Close the underlying writer
func (arvlog *ArvLogWriter) Close() (err error) {
	if arvlog.writeCloser != nil {
		err = arvlog.writeCloser.Close()
		arvlog.writeCloser = nil
	}
	return err
}
