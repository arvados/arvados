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
// ThrottledLogger.buf -> ThrottledLogger.flusher -> goWriter ->
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
}

// RFC3339Fixed is a fixed-width version of RFC3339 with microsecond precision,
// because the RFC3339Nano format isn't fixed width.
const RFC3339Fixed = "2006-01-02T15:04:05.000000Z07:00"

// RFC3339Timestamp return a RFC3339 formatted timestamp using RFC3339Fixed
func RFC3339Timestamp(now time.Time) string {
	return now.Format(RFC3339Fixed)
}

// Write to the internal buffer.  Prepend a timestamp to each line of the input
// data.
func (tl *ThrottledLogger) Write(p []byte) (n int, err error) {
	tl.Mutex.Lock()
	if tl.buf == nil {
		tl.buf = &bytes.Buffer{}
	}
	defer tl.Mutex.Unlock()

	now := tl.Timestamper(time.Now().UTC())
	sc := bufio.NewScanner(bytes.NewBuffer(p))
	for sc.Scan() {
		_, err = fmt.Fprintf(tl.buf, "%s %s\n", now, sc.Text())
	}
	return len(p), err
}

// Periodically check the current buffer; if not empty, send it on the
// channel to the goWriter goroutine.
func (tl *ThrottledLogger) flusher() {
	bufchan := make(chan *bytes.Buffer)
	bufterm := make(chan bool)

	// Use a separate goroutine for the actual write so that the writes are
	// actually initiated closer every 1s instead of every
	// 1s + (time to it takes to write).
	go goWriter(tl.writer, bufchan, bufterm)
	for {
		if !tl.stop {
			time.Sleep(1 * time.Second)
		}
		tl.Mutex.Lock()
		if tl.buf != nil && tl.buf.Len() > 0 {
			oldbuf := tl.buf
			tl.buf = nil
			tl.Mutex.Unlock()
			bufchan <- oldbuf
		} else if tl.stop {
			tl.Mutex.Unlock()
			break
		} else {
			tl.Mutex.Unlock()
		}
	}
	close(bufchan)
	<-bufterm
	tl.flusherDone <- true
}

// Receive buffers from a channel and send to the underlying Writer
func goWriter(writer io.Writer, c <-chan *bytes.Buffer, t chan<- bool) {
	for b := range c {
		writer.Write(b.Bytes())
	}
	t <- true
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
	alw := &ThrottledLogger{}
	alw.flusherDone = make(chan bool)
	alw.writer = writer
	alw.Logger = log.New(alw, "", 0)
	alw.Timestamper = RFC3339Timestamp
	go alw.flusher()
	return alw
}

// ArvLogWriter implements a writer that writes to each of a WriteCloser
// (typically CollectionFileWriter) and creates an API server log entry.
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
	lr := arvadosclient.Dict{"object_uuid": arvlog.UUID,
		"event_type": arvlog.loggingStream,
		"properties": map[string]string{"text": string(p)}}
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
