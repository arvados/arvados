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

func compareSlice(l1, l2 []int) bool {
	if len(l1) != len(l2) {
		return false
	}
	for i := range l1 {
		if l1[i] != l2[i] {
			return false
		}
	}
	return true
}

// Create a BlockWorkList, generate a list for it, and instantiate a worker.
func TestBlockWorkListReadWrite(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewBlockWorkList()
	b.ReplaceList(makeTestWorkList(input))

	output := make([]int, len(input))
	var i = 0
	for item := range b.NextItem {
		output[i] = item.Value.(int)
		i++
		if i >= len(output) {
			b.Close()
		}
	}

	if !compareSlice(output, input) {
		t.Fatalf("output %v does not match input %v\n", output, input)
	}
}

// Start a worker before the list has any input.
func TestBlockWorkListEarlyRead(t *testing.T) {
	var input = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}

	b := NewBlockWorkList()

	// Start a reader in a goroutine. The reader will block until the
	// block work list has been initialized.
	output := make([]int, len(input))
	done := make(chan int)
	go func() {
		var i = 0
		for item := range b.NextItem {
			output[i] = item.Value.(int)
			i++
			if i >= len(output) {
				b.Close()
			}
		}
		done <- 1
	}()

	// Feed the blocklist a new worklist, and wait for the worker to
	// finish.
	b.ReplaceList(makeTestWorkList(input))
	<-done

	if !compareSlice(output, input) {
		t.Fatalf("output %v does not match input %v\n", output, input)
	}
}

// Replace one active work list with another.
func TestBlockWorkListReplaceList(t *testing.T) {
	var input1 = []int{1, 1, 2, 3, 5, 8, 13, 21, 34}
	var input2 = []int{1, 4, 9, 16, 25, 36, 49, 64, 81}

	b := NewBlockWorkList()
	b.ReplaceList(makeTestWorkList(input1))

	// Read the first five elements from the work list.
	//
	output := make([]int, len(input1))
	for i := 0; i < 5; i++ {
		item := <-b.NextItem
		output[i] = item.Value.(int)
	}

	// Replace the work list and read the remaining elements.
	b.ReplaceList(makeTestWorkList(input2))
	i := 5
	for item := range b.NextItem {
		output[i] = item.Value.(int)
		i++
		if i >= len(output) {
			b.Close()
			break
		}
	}

	if !compareSlice(output[0:5], input1[0:5]) {
		t.Fatal("first half of output does not match")
	}
	if !compareSlice(output[5:], input2[0:4]) {
		t.Fatal("second half of output does not match")
	}
}
