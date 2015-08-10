package main

/* A WorkQueue is an asynchronous thread-safe queue manager.  It
   provides a channel from which items can be read off the queue, and
   permits replacing the contents of the queue at any time.

   The overall work flow for a WorkQueue is as follows:

     1. A WorkQueue is created with NewWorkQueue().  This
        function instantiates a new WorkQueue and starts a manager
        goroutine.  The manager listens on an input channel
        (manager.newlist) and an output channel (manager.NextItem).

     2. The manager first waits for a new list of requests on the
        newlist channel.  When another goroutine calls
        manager.ReplaceQueue(lst), it sends lst over the newlist
        channel to the manager.  The manager goroutine now has
        ownership of the list.

     3. Once the manager has this initial list, it listens on both the
        input and output channels for one of the following to happen:

          a. A worker attempts to read an item from the NextItem
             channel.  The manager sends the next item from the list
             over this channel to the worker, and loops.

          b. New data is sent to the manager on the newlist channel.
             This happens when another goroutine calls
             manager.ReplaceItem() with a new list.  The manager
             discards the current list, replaces it with the new one,
             and begins looping again.

          c. The input channel is closed.  The manager closes its
             output channel (signalling any workers to quit) and
             terminates.

   Tasks currently handled by WorkQueue:
     * the pull list
     * the trash list

   Example usage:

        // Any kind of user-defined type can be used with the
        // WorkQueue.
		type FrobRequest struct {
			frob string
		}

		// Make a work list.
		froblist := NewWorkQueue()

		// Start a concurrent worker to read items from the NextItem
		// channel until it is closed, deleting each one.
		go func(list WorkQueue) {
			for i := range list.NextItem {
				req := i.(FrobRequest)
				frob.Run(req)
			}
		}(froblist)

		// Set up a HTTP handler for PUT /frob
		router.HandleFunc(`/frob`,
			func(w http.ResponseWriter, req *http.Request) {
				// Parse the request body into a list.List
				// of FrobRequests, and give this list to the
				// frob manager.
				newfrobs := parseBody(req.Body)
				froblist.ReplaceQueue(newfrobs)
			}).Methods("PUT")

   Methods available on a WorkQueue:

		ReplaceQueue(list)
			Replaces the current item list with a new one.  The list
            manager discards any unprocessed items on the existing
            list and replaces it with the new one. If the worker is
            processing a list item when ReplaceQueue is called, it
            finishes processing before receiving items from the new
            list.
		Close()
			Shuts down the manager goroutine. When Close is called,
			the manager closes the NextItem channel.
*/

import "container/list"

type WorkQueue struct {
	getStatus chan WorkQueueStatus
	newlist   chan *list.List
	// Workers get work items by reading from this channel.
	NextItem <-chan interface{}
	// Each worker must send struct{}{} to ReportDone exactly once
	// for each work item received from NextItem, when it stops
	// working on that item (regardless of whether the work was
	// successful).
	ReportDone chan<- struct{}
}

type WorkQueueStatus struct {
	InProgress int
	Queued     int
}

// NewWorkQueue returns a new empty WorkQueue.
//
func NewWorkQueue() *WorkQueue {
	nextItem := make(chan interface{})
	reportDone := make(chan struct{})
	newList := make(chan *list.List)
	b := WorkQueue{
		getStatus:  make(chan WorkQueueStatus),
		newlist:    newList,
		NextItem:   nextItem,
		ReportDone: reportDone,
	}
	go func() {
		// Read new work lists from the newlist channel.
		// Reply to "status" and "get next item" queries by
		// sending to the getStatus and nextItem channels
		// respectively. Return when the newlist channel
		// closes.

		todo := &list.List{}
		status := WorkQueueStatus{}

		// When we're done, close the output channel; workers will
		// shut down next time they ask for new work.
		defer close(nextItem)
		defer close(b.getStatus)

		// nextChan and nextVal are both nil when we have
		// nothing to send; otherwise they are, respectively,
		// the nextItem channel and the next work item to send
		// to it.
		var nextChan chan interface{}
		var nextVal interface{}

		for newList != nil || status.InProgress > 0 {
			select {
			case p, ok := <-newList:
				if !ok {
					// Closed, stop receiving
					newList = nil
				}
				todo = p
				if todo == nil {
					todo = &list.List{}
				}
				status.Queued = todo.Len()
				if status.Queued == 0 {
					// Stop sending work
					nextChan = nil
					nextVal = nil
				} else {
					nextChan = nextItem
					nextVal = todo.Front().Value
				}
			case nextChan <- nextVal:
				todo.Remove(todo.Front())
				status.InProgress++
				status.Queued--
				if status.Queued == 0 {
					// Stop sending work
					nextChan = nil
					nextVal = nil
				} else {
					nextVal = todo.Front().Value
				}
			case <-reportDone:
				status.InProgress--
			case b.getStatus <- status:
			}
		}
	}()
	return &b
}

// ReplaceQueue abandons any work items left in the existing queue,
// and starts giving workers items from the given list. After giving
// it to ReplaceQueue, the caller must not read or write the given
// list.
//
func (b *WorkQueue) ReplaceQueue(list *list.List) {
	b.newlist <- list
}

// Close shuts down the manager and terminates the goroutine, which
// abandons any pending requests, but allows any pull request already
// in progress to continue.
//
// After Close, Status will return correct values, NextItem will be
// closed, and ReplaceQueue will panic.
//
func (b *WorkQueue) Close() {
	close(b.newlist)
}

// Status returns an up-to-date WorkQueueStatus reflecting the current
// queue status.
//
func (b *WorkQueue) Status() WorkQueueStatus {
	// If the channel is closed, we get the nil value of
	// WorkQueueStatus, which is an accurate description of a
	// finished queue.
	return <-b.getStatus
}
