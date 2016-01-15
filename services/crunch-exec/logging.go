package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"io"
	"log"
	"sync"
	"time"
)

type Timestamper func(t time.Time) string

// Logging plumbing:
//
// ThrottledLogger.Logger -> ThrottledLogger.Write ->
// ThrottledLogger.buf -> ThrottledLogger.flusher -> goWriter ->
// ArvLogWriter.Write -> CollectionFileWriter.Write | Api.Create
//
// For stdout/stderr CopyReaderToLog additionally runs as a goroutine to pull
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

// Builtin RFC3339Nano format isn't fixed width so
// provide our own with microsecond precision (same as API server).
const RFC3339Fixed = "2006-01-02T15:04:05.000000Z07:00"

func RFC3339Timestamp(now time.Time) string {
	return now.Format(RFC3339Fixed)
}

// Write to the internal buffer.  Prepend a timestamp to each line of the input
// data.
func (this *ThrottledLogger) Write(p []byte) (n int, err error) {
	this.Mutex.Lock()
	if this.buf == nil {
		this.buf = &bytes.Buffer{}
	}
	defer this.Mutex.Unlock()

	now := this.Timestamper(time.Now().UTC())
	sc := bufio.NewScanner(bytes.NewBuffer(p))
	for sc.Scan() {
		_, err = fmt.Fprintf(this.buf, "%s %s\n", now, sc.Text())
	}
	return len(p), err
}

// Periodically check the current buffer; if not empty, send it on the
// channel to the goWriter goroutine.
func (this *ThrottledLogger) flusher() {
	bufchan := make(chan *bytes.Buffer)
	bufterm := make(chan bool)

	// Use a separate goroutine for the actual write so that the writes are
	// actually initiated closer every 1s instead of every
	// 1s + (time to it takes to write).
	go goWriter(this.writer, bufchan, bufterm)
	for {
		if !this.stop {
			time.Sleep(1 * time.Second)
		}
		this.Mutex.Lock()
		if this.buf != nil && this.buf.Len() > 0 {
			oldbuf := this.buf
			this.buf = nil
			this.Mutex.Unlock()
			bufchan <- oldbuf
		} else if this.stop {
			this.Mutex.Unlock()
			break
		} else {
			this.Mutex.Unlock()
		}
	}
	close(bufchan)
	<-bufterm
	this.flusherDone <- true
}

// Receive buffers from a channel and send to the underlying Writer
func goWriter(writer io.Writer, c <-chan *bytes.Buffer, t chan<- bool) {
	for b := range c {
		writer.Write(b.Bytes())
	}
	t <- true
}

// Stop the flusher goroutine and wait for it to complete, then close the
// underlying Writer.
func (this *ThrottledLogger) Close() error {
	this.stop = true
	<-this.flusherDone
	return this.writer.Close()
}

const (
	MaxLogLine = 1 << 12 // Child stderr lines >4KiB will be split
)

// Goroutine to copy from a reader to a logger, with long line splitting.
func CopyReaderToLog(in io.Reader, logger *log.Logger, done chan<- bool) {
	reader := bufio.NewReaderSize(in, MaxLogLine)
	var prefix string
	for {
		line, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Print("error reading container log:", err)
		}
		var suffix string
		if isPrefix {
			suffix = "[...]"
		}
		logger.Print(prefix, string(line), suffix)
		// Set up prefix for following line
		if isPrefix {
			prefix = "[...]"
		} else {
			prefix = ""
		}
	}
	done <- true
}

// Create a new thottled logger that
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

// Implements a writer that writes to each of a WriteCloser (typically
// CollectionFileWriter) and creates an API server log entry.
type ArvLogWriter struct {
	Api           IArvadosClient
	Uuid          string
	loggingStream string
	writeCloser   io.WriteCloser
}

func (this *ArvLogWriter) Write(p []byte) (n int, err error) {
	// Write to the next writer in the chain (a file in Keep)
	var err1 error
	if this.writeCloser != nil {
		_, err1 = this.writeCloser.Write(p)
	}

	// write to API
	lr := arvadosclient.Dict{"object_uuid": this.Uuid,
		"event_type": this.loggingStream,
		"properties": map[string]string{"text": string(p)}}
	err2 := this.Api.Create("logs", lr, nil)

	if err1 != nil || err2 != nil {
		return 0, errors.New(fmt.Sprintf("%s ; %s", err1, err2))
	} else {
		return len(p), nil
	}

}

func (this *ArvLogWriter) Close() (err error) {
	if this.writeCloser != nil {
		err = this.writeCloser.Close()
		this.writeCloser = nil
	}
	return err
}
