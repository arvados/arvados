package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
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
	stopping chan struct{}
	stopped  chan struct{}
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
	for stopping := false; !stopping; {
		select {
		case <-tl.stopping:
			// flush tl.buf, then exit the loop
			stopping = true
		case <-ticker.C:
		}

		var ready *bytes.Buffer

		tl.Mutex.Lock()
		ready, tl.buf = tl.buf, nil
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
// (b) batches log messages and only calls the underlying Writer at most once
// per second.
func NewThrottledLogger(writer io.WriteCloser) *ThrottledLogger {
	tl := &ThrottledLogger{}
	tl.stopping = make(chan struct{})
	tl.stopped = make(chan struct{})
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

	// for rate limiting
	bytesLogged                  int64
	logThrottleResetTime         time.Time
	logThrottleLinesSoFar        int64
	logThrottleBytesSoFar        int64
	logThrottleBytesSkipped      int64
	logThrottleIsOpen            bool
	logThrottlePartialLineLastAt time.Time
	logThrottleFirstPartialLine  bool
	stderrBufToFlush             bytes.Buffer
	stderrFlushedAt              time.Time
}

func (arvlog *ArvLogWriter) Write(p []byte) (n int, err error) {
	// Write to the next writer in the chain (a file in Keep)
	var err1 error
	if arvlog.writeCloser != nil {
		_, err1 = arvlog.writeCloser.Write(p)
	}

	// write to API after checking rate limit
	crunchLogThrottlePeriod, err2 := arvlog.ArvClient.Discovery("crunchLogThrottlePeriod")
	crunchLogBytesPerEvent, err2 := arvlog.ArvClient.Discovery("crunchLogBytesPerEvent")
	crunchLogSecondsBetweenEvents, err2 := arvlog.ArvClient.Discovery("crunchLogSecondsBetweenEvents")
	if err2 != nil {
		return 0, fmt.Errorf("%s ; %s", err1, err2)
	}

	now := time.Now()
	bytesWritten := 0

	if now.After(arvlog.logThrottleResetTime) {
		// It has been more than throttle_period seconds since the last
		// checkpoint; so reset the throttle
		if arvlog.logThrottleBytesSkipped > 0 {
			arvlog.stderrBufToFlush.WriteString(fmt.Sprintf("%s Skipped %d bytes of log\n", RFC3339Timestamp(time.Now().UTC()), arvlog.logThrottleBytesSkipped))
		}

		arvlog.logThrottleResetTime = time.Now().Add(time.Second * time.Duration(int(crunchLogThrottlePeriod.(float64))))
		arvlog.logThrottleBytesSoFar = 0
		arvlog.logThrottleLinesSoFar = 0
		arvlog.logThrottleBytesSkipped = 0
		arvlog.logThrottleIsOpen = true
		arvlog.logThrottlePartialLineLastAt = time.Time{}
		arvlog.logThrottleFirstPartialLine = true
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
		_, msg, err2 := arvlog.rateLimit(line)
		if err2 != nil {
			return 0, fmt.Errorf("%s ; %s", err1, err2)
		}
		arvlog.stderrBufToFlush.WriteString(string(msg) + "\n")
	}

	if arvlog.stderrBufToFlush.Len() > int(crunchLogBytesPerEvent.(float64)) ||
		(time.Now().Sub(arvlog.stderrFlushedAt) >= time.Duration(int64(crunchLogSecondsBetweenEvents.(float64)))) {
		// write to API
		lr := arvadosclient.Dict{"log": arvadosclient.Dict{
			"object_uuid": arvlog.UUID,
			"event_type":  arvlog.loggingStream,
			"properties":  map[string]string{"text": arvlog.stderrBufToFlush.String()}}}
		err2 := arvlog.ArvClient.Create("logs", lr, nil)

		bytesWritten = arvlog.stderrBufToFlush.Len()
		arvlog.stderrBufToFlush = bytes.Buffer{}
		arvlog.stderrFlushedAt = time.Now()

		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("%s ; %s", err1, err2)
		}
	}

	return bytesWritten, nil
}

// Close the underlying writer
func (arvlog *ArvLogWriter) Close() (err error) {
	if arvlog.writeCloser != nil {
		err = arvlog.writeCloser.Close()
		arvlog.writeCloser = nil
	}
	return err
}

var lineRegexp = regexp.MustCompile(`^\S+ \S+ \d+ \d+ stderr (.*)`)

// Test for hard cap on total output and for log throttling. Returns whether
// the log line should go to output or not. Returns message if limit exceeded.
func (arvlog *ArvLogWriter) rateLimit(line []byte) (bool, []byte, error) {
	message := ""
	lineSize := int64(len(line))
	partialLine := false
	skipCounts := false
	if arvlog.logThrottleIsOpen {
		matches := lineRegexp.FindStringSubmatch(string(line))

		crunchLogPartialLineThrottlePeriod, err := arvlog.ArvClient.Discovery("crunchLogPartialLineThrottlePeriod")
		crunchLimitLogBytesPerJob, err := arvlog.ArvClient.Discovery("crunchLimitLogBytesPerJob")
		crunchLogThrottleBytes, err := arvlog.ArvClient.Discovery("crunchLogThrottleBytes")
		crunchLogThrottlePeriod, err := arvlog.ArvClient.Discovery("crunchLogThrottlePeriod")
		crunchLogThrottleLines, err := arvlog.ArvClient.Discovery("crunchLogThrottleLines")
		if err != nil {
			return false, []byte(""), err
		}

		if len(matches) == 2 && strings.HasPrefix(matches[1], "[...]") && strings.HasSuffix(matches[1], "[...]") {
			partialLine = true

			if time.Now().After(arvlog.logThrottlePartialLineLastAt.Add(time.Second * time.Duration(int(crunchLogPartialLineThrottlePeriod.(float64))))) {
				arvlog.logThrottlePartialLineLastAt = time.Now()
			} else {
				skipCounts = true
			}
		}

		if !skipCounts {
			arvlog.logThrottleLinesSoFar += 1
			arvlog.logThrottleBytesSoFar += lineSize
			arvlog.bytesLogged += lineSize
		}

		if arvlog.bytesLogged > int64(crunchLimitLogBytesPerJob.(float64)) {
			message = fmt.Sprintf("%s Exceeded log limit %d bytes (crunch_limit_log_bytes_per_job). Log will be truncated.", RFC3339Timestamp(time.Now().UTC()), int(crunchLimitLogBytesPerJob.(float64)))
			arvlog.logThrottleResetTime = time.Now().Add(time.Duration(365 * 24 * time.Hour))
			arvlog.logThrottleIsOpen = false

		} else if arvlog.logThrottleBytesSoFar > int64(crunchLogThrottleBytes.(float64)) {
			remainingTime := arvlog.logThrottleResetTime.Sub(time.Now())
			message = fmt.Sprintf("%s Exceeded rate %d bytes per %d seconds (crunch_log_throttle_bytes). Logging will be silenced for the next %d seconds.", RFC3339Timestamp(time.Now().UTC()), int(crunchLogThrottleBytes.(float64)), int(crunchLogThrottlePeriod.(float64)), remainingTime/time.Second)
			arvlog.logThrottleIsOpen = false

		} else if arvlog.logThrottleLinesSoFar > int64(crunchLogThrottleLines.(float64)) {
			remainingTime := arvlog.logThrottleResetTime.Sub(time.Now())
			message = fmt.Sprintf("%s Exceeded rate %d lines per %d seconds (crunch_log_throttle_lines), logging will be silenced for the next %d seconds.", RFC3339Timestamp(time.Now().UTC()), int(crunchLogThrottleLines.(float64)), int(crunchLogThrottlePeriod.(float64)), remainingTime/time.Second)
			arvlog.logThrottleIsOpen = false

		} else if partialLine && arvlog.logThrottleFirstPartialLine {
			arvlog.logThrottleFirstPartialLine = false
			message = fmt.Sprintf("%s Rate-limiting partial segments of long lines to one every %d seconds.", RFC3339Timestamp(time.Now().UTC()), int(crunchLogPartialLineThrottlePeriod.(float64)))

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
		return true, []byte(message), nil
	} else if partialLine {
		return false, line, nil
	} else {
		return arvlog.logThrottleIsOpen, line, nil
	}
}
