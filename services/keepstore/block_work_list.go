package main

/* A BlockWorkList is an asynchronous thread-safe queue manager.  It
   provides a channel from which items can be read off the queue, and
   permits replacing the contents of the queue at any time.

   The overall work flow for a BlockWorkList is as follows:

     1. A BlockWorkList is created with NewBlockWorkList().  This
        function instantiates a new BlockWorkList and starts a manager
        goroutine.  The manager listens on an input channel
        (manager.newlist) and an output channel (manager.NextItem).

     2. The manager first waits for a new list of requests on the
        newlist channel.  When another goroutine calls
        manager.ReplaceList(lst), it sends lst over the newlist
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

   Tasks currently handled by BlockWorkList:
     * the pull list
     * the trash list

   Example usage:

        // Any kind of user-defined type can be used with the
        // BlockWorkList.
		type FrobRequest struct {
			frob string
		}

		// Make a work list.
		froblist := NewBlockWorkList()

		// Start a concurrent worker to read items from the NextItem
		// channel until it is closed, deleting each one.
		go func(list BlockWorkList) {
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
				froblist.ReplaceList(newfrobs)
			}).Methods("PUT")

   Methods available on a BlockWorkList:

		ReplaceList(list)
			Replaces the current item list with a new one.  The list
            manager discards any unprocessed items on the existing
            list and replaces it with the new one. If the worker is
            processing a list item when ReplaceList is called, it
            finishes processing before receiving items from the new
            list.
		Close()
			Shuts down the manager goroutine. When Close is called,
			the manager closes the NextItem channel.
*/

import "container/list"

type BlockWorkList struct {
	items    *list.List
	newlist  chan *list.List
	NextItem chan interface{}
}

// NewBlockWorkList returns a new worklist, and launches a listener
// goroutine that waits for work and farms it out to workers.
//
func NewBlockWorkList() *BlockWorkList {
	b := BlockWorkList{
		items:    nil,
		newlist:  make(chan *list.List),
		NextItem: make(chan interface{}),
	}
	go b.listen()
	return &b
}

// ReplaceList sends a new list of pull requests to the manager goroutine.
// The manager will discard any outstanding pull list and begin
// working on the new list.
//
func (b *BlockWorkList) ReplaceList(list *list.List) {
	b.newlist <- list
}

// Close shuts down the manager and terminates the goroutine, which
// completes any pull request in progress and abandons any pending
// requests.
//
func (b *BlockWorkList) Close() {
	close(b.newlist)
}

// listen is run in a goroutine. It reads new pull lists from its
// input queue until the queue is closed.
// listen takes ownership of the list that is passed to it.
//
// Note that the routine does not ever need to access the list
// itself once the current_item has been initialized, so we do
// not bother to keep a pointer to the list. Because it is a
// doubly linked list, holding on to the current item will keep
// it from garbage collection.
//
func (b *BlockWorkList) listen() {
	var current_item *list.Element

	// When we're done, close the output channel to shut down any
	// workers.
	defer close(b.NextItem)

	for {
		// If the current list is empty, wait for a new list before
		// even checking if workers are ready.
		if current_item == nil {
			if p, ok := <-b.newlist; ok {
				current_item = p.Front()
			} else {
				// The channel was closed; shut down.
				return
			}
		}
		select {
		case p, ok := <-b.newlist:
			if ok {
				current_item = p.Front()
			} else {
				// The input channel is closed; time to shut down
				return
			}
		case b.NextItem <- current_item.Value:
			current_item = current_item.Next()
		}
	}
}
