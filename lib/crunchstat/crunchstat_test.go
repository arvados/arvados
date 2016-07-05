package crunchstat

import (
	"bufio"
	"io"
	"log"
	"os"
	"regexp"
	"testing"
)

func bufLogger() (*log.Logger, *bufio.Reader) {
	r, w := io.Pipe()
	logger := log.New(w, "", 0)
	return logger, bufio.NewReader(r)
}

func TestReadAllOrWarnFail(t *testing.T) {
	logger, rcv := bufLogger()
	rep := Reporter{Logger: logger}

	done := make(chan bool)
	var msg []byte
	var err error
	go func() {
		msg, err = rcv.ReadBytes('\n')
		close(done)
	}()
	{
		// The special file /proc/self/mem can be opened for
		// reading, but reading from byte 0 returns an error.
		f, err := os.Open("/proc/self/mem")
		if err != nil {
			t.Fatalf("Opening /proc/self/mem: %s", err)
		}
		if x, err := rep.readAllOrWarn(f); err == nil {
			t.Fatalf("Expected error, got %v", x)
		}
	}
	<-done
	if err != nil {
		t.Fatal(err)
	} else if matched, err := regexp.MatchString("^read /proc/self/mem: .*", string(msg)); err != nil || !matched {
		t.Fatalf("Expected error message about unreadable file, got \"%s\"", msg)
	}
}

func TestReadAllOrWarnSuccess(t *testing.T) {
	rep := Reporter{Logger: log.New(os.Stderr, "", 0)}

	f, err := os.Open("./crunchstat_test.go")
	if err != nil {
		t.Fatalf("Opening ./crunchstat_test.go: %s", err)
	}
	data, err := rep.readAllOrWarn(f)
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	if matched, err := regexp.MatchString("^package crunchstat\n", string(data)); err != nil || !matched {
		t.Fatalf("data failed regexp: err %v, matched %v", err, matched)
	}
}
