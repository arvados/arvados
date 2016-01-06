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

type ThrottledLogger struct {
	*log.Logger
	buf *bytes.Buffer
	sync.Mutex
	writer      io.WriteCloser
	stop        bool
	flusherDone chan bool
	Timestamper
}

func RFC3339Timestamp(now time.Time) string {
	// return now.Format(time.RFC3339Nano)
	// Builtin RFC3339Nano format isn't fixed width so
	// provide our own.

	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%09dZ",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		now.Nanosecond())

}

func (this *ThrottledLogger) Write(p []byte) (n int, err error) {
	this.Mutex.Lock()
	if this.buf == nil {
		this.buf = &bytes.Buffer{}
	}
	defer this.Mutex.Unlock()

	now := time.Now().UTC()
	_, err = fmt.Fprintf(this.buf, "%s %s", this.Timestamper(now), p)
	return len(p), err
}

func (this *ThrottledLogger) Stop() {
	this.stop = true
	<-this.flusherDone
	this.writer.Close()
}

func goWriter(writer io.Writer, c <-chan *bytes.Buffer, t chan<- bool) {
	for b := range c {
		writer.Write(b.Bytes())
	}
	t <- true
}

func (this *ThrottledLogger) flusher() {
	bufchan := make(chan *bytes.Buffer)
	bufterm := make(chan bool)
	go goWriter(this.writer, bufchan, bufterm)
	for {
		if !this.stop {
			time.Sleep(1 * time.Second)
		}
		this.Mutex.Lock()
		if this.buf != nil && this.buf.Len() > 0 {
			bufchan <- this.buf
			this.buf = nil
		} else if this.stop {
			this.Mutex.Unlock()
			break
		}
		this.Mutex.Unlock()
	}
	close(bufchan)
	<-bufterm
	this.flusherDone <- true
}

const (
	MaxLogLine = 1 << 12 // Child stderr lines >4KiB will be split
)

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

func NewThrottledLogger(writer io.WriteCloser) *ThrottledLogger {
	alw := &ThrottledLogger{}
	alw.flusherDone = make(chan bool)
	alw.writer = writer
	alw.Logger = log.New(alw, "", 0)
	alw.Timestamper = RFC3339Timestamp
	go alw.flusher()
	return alw
}

type ArvLogWriter struct {
	Api           IArvadosClient
	Uuid          string
	loggingStream string
	io.WriteCloser
}

func (this *ArvLogWriter) Write(p []byte) (n int, err error) {
	var err1 error
	if this.WriteCloser != nil {
		_, err1 = this.WriteCloser.Write(p)
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
	if this.WriteCloser != nil {
		err = this.WriteCloser.Close()
		this.WriteCloser = nil
	}
	return err
}
