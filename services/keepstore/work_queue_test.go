package main

import (
	"container/list"
	"testing"
)

func makeTestWorkList(ary []int) *list.List {
	l := list.New()
	for _, n := range ary {
		l.PushBack(n)
	}
	return l
}

func expectChannelEmpty(t *testing.T, c <-chan interface{}) {
	select {
	case item := <-c:
		t.Fatalf("Received value (%v) from channel that we expected to be empty", item)
	default:
		// no-op
	}
}

func expectChannelNotEmpty(t *testing.T, c <-chan interface{}) {
	if item, ok := <-c; !ok {
		t.Fatal("expected data on a closed channel")
	} else if item == nil {
		t.Fatal("expected data on an empty channel")
	}
}

func expectChannelClosed(t *testing.T, c <-chan interface{}) {
	received, ok := <-c
	if ok {
		t.Fatalf("Expected channel to be closed, but received %v instead", received)
	}
}

func expectFromChannel(t *testing.T, c <-chan interface{}, expected []int) {
	for i := range expected {
		actual, ok := <-c
		t.Logf("received %v", actual)
		if !ok {
			t.Fatalf("Expected %v but channel was closed after receiving the first %d elements correctly.", expected, i)
		} else if actual.(int) != expected[i] {
			t.Fatalf("Expected %v but received '%v' after receiving the first %d elements correctly.", expected[i], actual, i)
		}
	}
}

// Create a WorkQueue, generate a list for it, and instantiate a worker.
func TestWorkQueueReadWrite(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewWorkQueue()
	b.ReplaceQueue(makeTestWorkList(input))

	expectFromChannel(t, b.NextItem, input)
	expectChannelEmpty(t, b.NextItem)
	b.Close()
}

// Start a worker before the list has any input.
func TestWorkQueueEarlyRead(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewWorkQueue()

	// First, demonstrate that nothing is available on the NextItem
	// channel.
	expectChannelEmpty(t, b.NextItem)

	// Start a reader in a goroutine. The reader will block until the
	// block work list has been initialized.
	//
	done := make(chan int)
	go func() {
		expectFromChannel(t, b.NextItem, input)
		b.Close()
		done <- 1
	}()

	// Feed the blocklist a new worklist, and wait for the worker to
	// finish.
	b.ReplaceQueue(makeTestWorkList(input))
	<-done

	expectChannelClosed(t, b.NextItem)
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
	sendmore := make(chan int)
	done := make(chan int)
	go func() {
		expectFromChannel(t, b.NextItem, inputBeforeBlock)

		// Confirm that the channel is empty, so a subsequent read
		// on it will block.
		expectChannelEmpty(t, b.NextItem)

		// Signal that we're ready for more input.
		sendmore <- 1
		expectFromChannel(t, b.NextItem, inputAfterBlock)
		b.Close()
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
	expectFromChannel(t, b.NextItem, firstInput[0:5])
	expectChannelNotEmpty(t, b.NextItem)

	// Replace the work list and read five more elements.
	// The old list should have been discarded and all new
	// elements come from the new list.
	b.ReplaceQueue(makeTestWorkList(replaceInput))
	expectFromChannel(t, b.NextItem, replaceInput[0:5])

	b.Close()
}
