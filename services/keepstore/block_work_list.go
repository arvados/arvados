package main

/* A BlockWorkList concurrently processes blocks needing attention.

   Tasks currently handled by BlockWorkList:
     * the pull list
     * the trash list

   A BlockWorkList is instantiated with NewBlockWorkList(), which
   launches a manager in a goroutine.  The manager listens on a
   channel for data to be assigned to it via the ReplaceList() method.

   A worker gets items to process from a BlockWorkList by reading the
   NextItem channel.  The list manager continuously writes items to
   this channel.

   Example (simplified) implementation of a trash collector:

		type DeleteRequest struct {
			hash string
			age time.Time
		}

		// Make a work list.
		trashList := NewBlockWorkList()

		// Start a concurrent worker to read items from the NextItem
		// channel until it is closed, deleting each one.
		go func(list BlockWorkList) {
			for i := range list.NextItem {
				req := i.(DeleteRequest)
				if time.Now() > req.age {
					deleteBlock(req.hash)
				}
			}
		}(trashList)

		// Set up a HTTP handler for PUT /trash
		router.HandleFunc(`/trash`,
			func(w http.ResponseWriter, req *http.Request) {
				// Parse the request body into a list.List
				// of DeleteRequests, and give this list to the
				// trash collector.
				trash := parseBody(req.Body)
				trashList.ReplaceList(trash)
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
			Shuts down the manager and the worker cleanly.
*/

import "container/list"

type BlockWorkList struct {
	items    *list.List
	newlist  chan *list.List
	NextItem chan *list.Element
}

// NewBlockWorkList returns a new worklist, and launches a listener
// goroutine that waits for work and farms it out to workers.
//
func NewBlockWorkList() *BlockWorkList {
	b := BlockWorkList{
		items:    nil,
		newlist:  make(chan *list.List),
		NextItem: make(chan *list.Element),
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
func (b *BlockWorkList) listen() {
	var (
		current_list *list.List
		current_item *list.Element
	)

	// When we're done, close the output channel to shut down any
	// workers.
	defer close(b.NextItem)

	for {
		// If the current list is empty, wait for a new list before
		// even checking if workers are ready.
		if current_item == nil {
			if p, ok := <-b.newlist; ok {
				current_list = p
			} else {
				// The channel was closed; shut down.
				return
			}
			current_item = current_list.Front()
		}
		select {
		case p, ok := <-b.newlist:
			if ok {
				current_list = p
				current_item = current_list.Front()
			} else {
				// The input channel is closed; time to shut down
				return
			}
		case b.NextItem <- current_item:
			current_item = current_item.Next()
		}
	}
}
