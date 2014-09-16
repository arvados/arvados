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

// peek returns the next item available from the channel, or
// nil if the channel is empty or closed.
func peek(c <-chan interface{}) interface{} {
	select {
	case item := <-c:
		return item
	default:
		return nil
	}
}

// Create a BlockWorkList, generate a list for it, and instantiate a worker.
func TestBlockWorkListReadWrite(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewBlockWorkList()
	b.ReplaceList(makeTestWorkList(input))

	var i = 0
	for item := range b.NextItem {
		if item.(int) != input[i] {
			t.Fatalf("expected %d, got %d", input[i], item.(int))
		}
		i++
		if i >= len(input) {
			break
		}
	}

	if item := peek(b.NextItem); item != nil {
		t.Fatalf("unexpected output %v", item)
	}
}

// Start a worker before the list has any input.
func TestBlockWorkListEarlyRead(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewBlockWorkList()

	// First, demonstrate that nothing is available on the NextItem
	// channel.
	if item := peek(b.NextItem); item != nil {
		t.Fatalf("unexpected output %v", item)
	}

	// Start a reader in a goroutine. The reader will block until the
	// block work list has been initialized.
	// Note that the worker closes itself: once it has read as many
	// elements as it expects, it calls b.Close(), which causes the
	// manager to close the b.NextItem channel.
	//
	done := make(chan int)
	go func() {
		var i = 0
		defer func() { done <- 1 }()
		for item := range b.NextItem {
			if item.(int) != input[i] {
				t.Fatalf("expected %d, got %d", input[i], item.(int))
			}
			i++
			if i >= len(input) {
				b.Close()
			}
		}
	}()

	// Feed the blocklist a new worklist, and wait for the worker to
	// finish.
	b.ReplaceList(makeTestWorkList(input))
	<-done

	if item := peek(b.NextItem); item != nil {
		t.Fatalf("unexpected output %v", item)
	}
}

// Show that a reader may block when the manager's list is exhausted,
// and that the reader resumes automatically when new data is
// available.
func TestBlockWorkListReaderBlocks(t *testing.T) {
	var input = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b := NewBlockWorkList()
	sendmore := make(chan int)
	done := make(chan int)
	go func() {
		i := 0
		for item := range b.NextItem {
			if item.(int) != input[i] {
				t.Fatalf("expected %d, got %d", input[i], item.(int))
			}
			i++
			if i == 5 {
				sendmore <- 1
			}
			if i == 10 {
				b.Close()
			}
		}
		done <- 1
	}()

	// Write a slice of the first five elements and wait for a signal
	// from the reader.
	b.ReplaceList(makeTestWorkList(input[0:5]))
	<-sendmore

	// Confirm that no more data is available on the NextItem channel
	// (and therefore any readers are blocked) before writing the
	// final five elements.
	if item := peek(b.NextItem); item != nil {
		t.Fatalf("unexpected output %v", item)
	}
	b.ReplaceList(makeTestWorkList(input[5:]))

	// Wait for the reader to complete.
	<-done
}

// Replace one active work list with another.
func TestBlockWorkListReplaceList(t *testing.T) {
	var input1 = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}
	var input2 = []int{1, 4, 9, 16, 25, 36, 49, 64, 81}

	b := NewBlockWorkList()
	b.ReplaceList(makeTestWorkList(input1))

	// Read the first five elements from the work list.
	//
	for i := 0; i < 5; i++ {
		item := <-b.NextItem
		if item.(int) != input1[i] {
			t.Fatalf("expected %d, got %d", input1[i], item.(int))
		}
	}

	// Replace the work list and read five more elements.
	b.ReplaceList(makeTestWorkList(input2))
	for i := 0; i < 5; i++ {
		item := <-b.NextItem
		if item.(int) != input2[i] {
			t.Fatalf("expected %d, got %d", input2[i], item.(int))
		}
	}
}
