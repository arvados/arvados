package main

import (
	"os"
	"regexp"
	"testing"
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
