package pull_list

/* The pull_list package manages a list of pull requests sent
   by Data Manager.

   The interface is:

   pull_list.NewManager() creates and returns a pull_list.Manager. A
   listener runs in a goroutine, waiting for new requests on its input
   channels.

   pull_list.SetList() assigns a new pull list to the manager. Any
   existing list is discarded.

   pull_list.GetList() reports the manager's current pull list.

   pull_list.Close() shuts down the pull list manager.
*/

type PullRequest struct {
	Locator string
	Servers []string
}

type Manager struct {
	setlist chan []PullRequest // input channel for setting new lists
	getlist chan []PullRequest // output channel for getting existing list
}

// NewManager returns a new Manager object.  It launches a goroutine that
// waits for pull requests.
//
func NewManager() *Manager {
	r := Manager{
		make(chan []PullRequest),
		make(chan []PullRequest),
	}
	go r.listen()
	return &r
}

// SetList sends a new list of pull requests to the manager goroutine.
// The manager will discard any outstanding pull list and begin
// working on the new list.
//
func (r *Manager) SetList(pr []PullRequest) {
	r.setlist <- pr
}

// GetList reports the contents of the current pull list.
func (r *Manager) GetList() []PullRequest {
	return <-r.getlist
}

// Close shuts down the manager and terminates the goroutine, which
// completes any pull request in progress and abandons any pending
// requests.
//
func (r *Manager) Close() {
	close(r.setlist)
}

// listen is run in a goroutine. It reads new pull lists from its
// input queue until the queue is closed.
func (r *Manager) listen() {
	var current []PullRequest
	for {
		select {
		case p, ok := <-r.setlist:
			if ok {
				current = p
			} else {
				// The input channel is closed; time to shut down
				close(r.getlist)
				return
			}
		case r.getlist <- current:
			// no-op
		}
	}
}
