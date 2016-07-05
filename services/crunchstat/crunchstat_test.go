package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"math/rand"
	"testing"
	"time"
)

// Test that CopyPipeToChildLog works even on lines longer than
// bufio.MaxScanTokenSize.
func TestCopyPipeToChildLogLongLines(t *testing.T) {
	logger, logBuf := bufLogger()

	pipeIn, pipeOut := io.Pipe()
	copied := make(chan bool)
	go func() {
		copyPipeToChildLog(pipeIn, logger)
		close(copied)
	}()

	sentBytes := make([]byte, bufio.MaxScanTokenSize+MaxLogLine+(1<<22))
	go func() {
		pipeOut.Write([]byte("before\n"))

		for i := range sentBytes {
			// Some bytes that aren't newlines:
			sentBytes[i] = byte((rand.Int() & 0xff) | 0x80)
		}
		sentBytes[len(sentBytes)-1] = '\n'
		pipeOut.Write(sentBytes)

		pipeOut.Write([]byte("after"))
		pipeOut.Close()
	}()

	if before, err := logBuf.ReadBytes('\n'); err != nil || string(before) != "before\n" {
		t.Fatalf("\"before\n\" not received (got \"%s\", %s)", before, err)
	}

	var receivedBytes []byte
	done := false
	for !done {
		line, err := logBuf.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if len(line) >= 5 && string(line[0:5]) == "[...]" {
			if receivedBytes == nil {
				t.Fatal("Beginning of line reported as continuation")
			}
			line = line[5:]
		}
		if len(line) >= 6 && string(line[len(line)-6:]) == "[...]\n" {
			line = line[:len(line)-6]
		} else {
			done = true
		}
		receivedBytes = append(receivedBytes, line...)
	}
	if bytes.Compare(receivedBytes, sentBytes) != 0 {
		t.Fatalf("sent %d bytes, got %d different bytes", len(sentBytes), len(receivedBytes))
	}

	if after, err := logBuf.ReadBytes('\n'); err != nil || string(after) != "after\n" {
		t.Fatalf("\"after\n\" not received (got \"%s\", %s)", after, err)
	}

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	case <-copied:
		// Done.
	}
}

func bufLogger() (*log.Logger, *bufio.Reader) {
	r, w := io.Pipe()
	logger := log.New(w, "", 0)
	return logger, bufio.NewReader(r)
}
