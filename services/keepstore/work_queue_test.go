package main

import (
	"container/list"
	"runtime"
	"testing"
	"time"
)

type fatalfer interface {
	Fatalf(string, ...interface{})
}

func makeTestWorkList(ary []int) *list.List {
	l := list.New()
	for _, n := range ary {
		l.PushBack(n)
	}
	return l
}

func expectChannelEmpty(t fatalfer, c <-chan interface{}) {
	select {
	case item, ok := <-c:
		if ok {
			t.Fatalf("Received value (%+v) from channel that we expected to be empty", item)
		}
	default:
	}
}

func expectChannelNotEmpty(t fatalfer, c <-chan interface{}) interface{} {
	select {
	case item, ok := <-c:
		if !ok {
			t.Fatalf("expected data on a closed channel")
		}
		return item
	case <-time.After(time.Second):
		t.Fatalf("expected data on an empty channel")
		return nil
	}
}

func expectChannelClosedWithin(t fatalfer, timeout time.Duration, c <-chan interface{}) {
	select {
	case received, ok := <-c:
		if ok {
			t.Fatalf("Expected channel to be closed, but received %+v instead", received)
		}
	case <-time.After(timeout):
		t.Fatalf("Expected channel to be closed, but it is still open after %v", timeout)
	}
}

func doWorkItems(t fatalfer, q *WorkQueue, expected []int) {
	for i := range expected {
		actual, ok := <-q.NextItem
		if !ok {
			t.Fatalf("Expected %+v but channel was closed after receiving %+v as expected.", expected, expected[:i])
		}
		q.DoneItem <- struct{}{}
		if actual.(int) != expected[i] {
			t.Fatalf("Expected %+v but received %+v after receiving %+v as expected.", expected[i], actual, expected[:i])
		}
	}
}

func expectEqualWithin(t fatalfer, timeout time.Duration, expect interface{}, f func() interface{}) {
	ok := make(chan struct{})
	giveup := false
	go func() {
		for f() != expect && !giveup {
			time.Sleep(time.Millisecond)
		}
		close(ok)
	}()
	select {
	case <-ok:
	case <-time.After(timeout):
		giveup = true
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("Still getting %+v, timed out waiting for %+v\n%s:%d", f(), expect, file, line)
	}
}

func expectQueued(t fatalfer, b *WorkQueue, expectQueued int) {
	if l := b.Status().Queued; l != expectQueued {
		t.Fatalf("Got Queued==%d, expected %d", l, expectQueued)
	}
}

func TestWorkQueueDoneness(t *testing.T) {
	b := NewWorkQueue()
	defer b.Close()
	b.ReplaceQueue(makeTestWorkList([]int{1, 2, 3}))
	expectQueued(t, b, 3)
	gate := make(chan struct{})
	go func() {
		<-gate
		for _ = range b.NextItem {
			<-gate
			time.Sleep(time.Millisecond)
			b.DoneItem <- struct{}{}
		}
	}()
	expectEqualWithin(t, time.Second, 0, func() interface{} { return b.Status().InProgress })
	b.ReplaceQueue(makeTestWorkList([]int{4, 5, 6}))
	for i := 1; i <= 3; i++ {
		gate <- struct{}{}
		expectEqualWithin(t, time.Second, 3-i, func() interface{} { return b.Status().Queued })
		expectEqualWithin(t, time.Second, 1, func() interface{} { return b.Status().InProgress })
	}
	close(gate)
	expectEqualWithin(t, time.Second, 0, func() interface{} { return b.Status().InProgress })
	expectChannelEmpty(t, b.NextItem)
}

// Create a WorkQueue, generate a list for it, and instantiate a worker.
func TestWorkQueueReadWrite(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewWorkQueue()
	expectQueued(t, b, 0)

	b.ReplaceQueue(makeTestWorkList(input))
	expectQueued(t, b, len(input))

	doWorkItems(t, b, input)
	expectChannelEmpty(t, b.NextItem)
	b.Close()
}

// Start a worker before the list has any input.
func TestWorkQueueEarlyRead(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewWorkQueue()
	defer b.Close()

	// First, demonstrate that nothing is available on the NextItem
	// channel.
	expectChannelEmpty(t, b.NextItem)

	// Start a reader in a goroutine. The reader will block until the
	// block work list has been initialized.
	//
	done := make(chan int)
	go func() {
		doWorkItems(t, b, input)
		done <- 1
	}()

	// Feed the blocklist a new worklist, and wait for the worker to
	// finish.
	b.ReplaceQueue(makeTestWorkList(input))
	<-done
	expectQueued(t, b, 0)
}

// After Close(), NextItem closes, work finishes, then stats return zero.
func TestWorkQueueClose(t *testing.T) {
	b := NewWorkQueue()
	input := []int{1, 2, 3, 4, 5, 6, 7, 8}
	mark := make(chan struct{})
	go func() {
		<-b.NextItem
		mark <- struct{}{}
		<-mark
		b.DoneItem <- struct{}{}
	}()
	b.ReplaceQueue(makeTestWorkList(input))
	// Wait for worker to take item 1
	<-mark
	b.Close()
	expectEqualWithin(t, time.Second, 1, func() interface{} { return b.Status().InProgress })
	// Tell worker to report done
	mark <- struct{}{}
	expectEqualWithin(t, time.Second, 0, func() interface{} { return b.Status().InProgress })
	expectChannelClosedWithin(t, time.Second, b.NextItem)
}

// Show that a reader may block when the manager's list is exhausted,
// and that the reader resumes automatically when new data is
// available.
func TestWorkQueueReaderBlocks(t *testing.T) {
	var (
		inputBeforeBlock = []int{1, 2, 3, 4, 5}
		inputAfterBlock  = []int{6, 7, 8, 9, 10}
	)

	b := NewWorkQueue()
	defer b.Close()
	sendmore := make(chan int)
	done := make(chan int)
	go func() {
		doWorkItems(t, b, inputBeforeBlock)

		// Confirm that the channel is empty, so a subsequent read
		// on it will block.
		expectChannelEmpty(t, b.NextItem)

		// Signal that we're ready for more input.
		sendmore <- 1
		doWorkItems(t, b, inputAfterBlock)
		done <- 1
	}()

	// Write a slice of the first five elements and wait for the
	// reader to signal that it's ready for us to send more input.
	b.ReplaceQueue(makeTestWorkList(inputBeforeBlock))
	<-sendmore

	b.ReplaceQueue(makeTestWorkList(inputAfterBlock))

	// Wait for the reader to complete.
	<-done
}

// Replace one active work list with another.
func TestWorkQueueReplaceQueue(t *testing.T) {
	var firstInput = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}
	var replaceInput = []int{1, 4, 9, 16, 25, 36, 49, 64, 81}

	b := NewWorkQueue()
	b.ReplaceQueue(makeTestWorkList(firstInput))

	// Read just the first five elements from the work list.
	// Confirm that the channel is not empty.
	doWorkItems(t, b, firstInput[0:5])
	expectChannelNotEmpty(t, b.NextItem)

	// Replace the work list and read five more elements.
	// The old list should have been discarded and all new
	// elements come from the new list.
	b.ReplaceQueue(makeTestWorkList(replaceInput))
	doWorkItems(t, b, replaceInput[0:5])

	b.Close()
}
