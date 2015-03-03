package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"math/rand"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestReadAllOrWarnFail(t *testing.T) {
	rcv := captureLogs()
	defer uncaptureLogs()
	go func() {
		// The special file /proc/self/mem can be opened for
		// reading, but reading from byte 0 returns an error.
		f, err := os.Open("/proc/self/mem")
		if err != nil {
			t.Fatalf("Opening /proc/self/mem: %s", err)
		}
		if x, err := ReadAllOrWarn(f); err == nil {
			t.Fatalf("Expected error, got %v", x)
		}
	}()
	if msg, err := rcv.ReadBytes('\n'); err != nil {
		t.Fatal(err)
	} else if matched, err := regexp.MatchString("^crunchstat: .*error.*", string(msg)); err != nil || !matched {
		t.Fatalf("Expected error message about unreadable file, got \"%s\"", msg)
	}
}

func TestReadAllOrWarnSuccess(t *testing.T) {
	f, err := os.Open("./crunchstat_test.go")
	if err != nil {
		t.Fatalf("Opening ./crunchstat_test.go: %s", err)
	}
	data, err := ReadAllOrWarn(f)
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	if matched, err := regexp.MatchString("^package main\n", string(data)); err != nil || !matched {
		t.Fatalf("data failed regexp: %s", err)
	}
}

// Test that CopyPipeToChildLog works even on lines longer than
// bufio.MaxScanTokenSize.
func TestCopyPipeToChildLogLongLines(t *testing.T) {
	rcv := captureLogs()
	defer uncaptureLogs()

	control := make(chan bool)
	pipeIn, pipeOut := io.Pipe()
	go CopyPipeToChildLog(pipeIn, control)

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

	if before, err := rcv.ReadBytes('\n'); err != nil || string(before) != "before\n" {
		t.Fatalf("\"before\n\" not received (got \"%s\", %s)", before, err)
	}

	var receivedBytes []byte
	done := false
	for !done {
		line, err := rcv.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if len(line) >= 5 && string(line[0:5]) == "[...]" {
			if receivedBytes == nil {
				t.Fatal("Beginning of line reported as continuation")
			}
			line = line[5:]
		}
		if len(line) >= 6 && string(line[len(line)-6:len(line)]) == "[...]\n" {
			line = line[:len(line)-6]
		} else {
			done = true
		}
		receivedBytes = append(receivedBytes, line...)
	}
	if bytes.Compare(receivedBytes, sentBytes) != 0 {
		t.Fatalf("sent %d bytes, got %d different bytes", len(sentBytes)+1, len(receivedBytes))
	}

	if after, err := rcv.ReadBytes('\n'); err != nil || string(after) != "after\n" {
		t.Fatal("\"after\n\" not received (got \"%s\", %s)", after, err)
	}

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	case <-control:
		// Done.
	}
}

func captureLogs() *bufio.Reader {
	// Send childLog to our bufio reader instead of stderr
	stderrIn, stderrOut := io.Pipe()
	childLog = log.New(stderrOut, "", 0)
	statLog = log.New(stderrOut, "crunchstat: ", 0)
	return bufio.NewReader(stderrIn)
}

func uncaptureLogs() {
	childLog = log.New(os.Stderr, "", 0)
	statLog = log.New(os.Stderr, "crunchstat: ", 0)
}
