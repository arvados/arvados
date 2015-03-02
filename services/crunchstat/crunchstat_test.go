package main

import (
	"bufio"
	"bytes"
	"io"
	"math/rand"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestReadAllOrWarnFail(t *testing.T) {
	logChan = make(chan string)
	go func() {
		defer close(logChan)
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
	if _, ok := <-logChan; !ok {
		t.Fatalf("Expected error message about nonexistent file")
	}
	if msg, ok := <-logChan; ok {
		t.Fatalf("Expected channel to close, got %s", msg)
	}
}

func TestReadAllOrWarnSuccess(t *testing.T) {
	logChan = make(chan string)
	go func() {
		defer close(logChan)
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
	}()
	if msg, ok := <-logChan; ok {
		t.Fatalf("Expected channel to close, got %s", msg)
	}
}

// Test that CopyPipeToChan works even on lines longer than
// bufio.MaxScanTokenSize.
func TestCopyPipeToChanLongLines(t *testing.T) {
	logChan := make(chan string)
	control := make(chan bool)

	pipeIn, pipeOut := io.Pipe()
	go CopyPipeToChan(pipeIn, logChan, control)

	sentBytes := make([]byte, bufio.MaxScanTokenSize + (1 << 22))
	go func() {
		for i := range sentBytes {
			// Some bytes that aren't newlines:
			sentBytes[i] = byte((rand.Int() & 0xff) | 0x80)
		}
		pipeOut.Write([]byte("before\n"))
		pipeOut.Write(sentBytes)
		pipeOut.Write([]byte("\nafter\n"))
		pipeOut.Close()
	}()

	if before := <-logChan; before != "before" {
		t.Fatalf("\"before\" not received (got \"%s\")", before)
	}
	receivedString := <-logChan
	receivedBytes := []byte(receivedString)
	if bytes.Compare(receivedBytes, sentBytes) != 0 {
		t.Fatalf("sent %d bytes, got %d different bytes", len(sentBytes), len(receivedBytes))
	}
	if after := <-logChan; after != "after" {
		t.Fatal("\"after\" not received")
	}
	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	case <-control:
		// Done.
	}
}
