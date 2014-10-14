package main

import (
	"regexp"
	"testing"
)

func TestOpenAndReadAllFail(t *testing.T) {
	log_chan := make(chan string)
	go func() {
		defer close(log_chan)
		if x, err := OpenAndReadAll("/nonexistent/file", log_chan); err == nil {
			t.Fatalf("Expected error, got %v", x)
		}
	}()
	if _, ok := <-log_chan; !ok {
		t.Fatalf("Expected error message about nonexistent file")
	}
	if msg, ok := <-log_chan; ok {
		t.Fatalf("Expected channel to close, got %s", msg)
	}
}

func TestOpenAndReadAllSuccess(t *testing.T) {
	log_chan := make(chan string)
	go func() {
		defer close(log_chan)
		data, err := OpenAndReadAll("./crunchstat_test.go", log_chan)
		if err != nil {
			t.Fatalf("got error %s", err)
		}
		if matched, err := regexp.MatchString("^package main\n", string(data)); err != nil || !matched {
			t.Fatalf("data failed regexp: %s", err)
		}
	}()
	if msg, ok := <-log_chan; ok {
		t.Fatalf("Expected channel to close, got %s", msg)
	}
}
