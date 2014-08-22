package replicator

/* The Keep replicator package fulfills replication pull requests sent
   by Data Manager.

   The interface is:

   replicator.New() launches a replication goroutine and returns the
   new Replicator object.

   replicator.Pull() assigns a new pull list to the goroutine.

   replicator.Dump() reports the goroutine's current pull list.

   replicator.Close() shuts down the replicator.
*/

type PullRequest struct {
	Locator string
	Servers []string
}

type Replicator struct {
	queue chan []PullRequest
	dump  chan []PullRequest
}

// New returns a new Replicator object.  It launches a goroutine that
// waits for pull requests.
//
func New() *Replicator {
	r := Replicator{
		make(chan []PullRequest),
		make(chan []PullRequest),
	}
	go r.listen()
	return &r
}

// Pull sends a new list of pull requests to the replicator goroutine.
// The replicator will discard any outstanding pull requests and begin
// working on the new list.
//
func (r *Replicator) Pull(pr []PullRequest) {
	r.queue <- pr
}

// Dump reports the contents of the current pull list.
func (r *Replicator) Dump() []PullRequest {
	return <-r.dump
}

// Close shuts down the replicator and terminates the goroutine, which
// completes any pull request in progress and abandons any pending
// requests.
//
func (r *Replicator) Close() {
	close(r.queue)
}

// listen is run in a goroutine. It reads new pull lists from its
// input queue until the queue is closed.
func (r *Replicator) listen() {
	var current []PullRequest
	for {
		select {
		case p, ok := <-r.queue:
			if ok {
				current = p
			} else {
				// The input channel is closed; time to shut down
				close(r.dump)
				return
			}
		case r.dump <- current:
			// no-op
		}
	}
}
