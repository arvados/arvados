package trash

/* The Keep trash collector processes the trash list sent
   by Data Manager.

   The interface is:

   trash.New() launches a trash collection goroutine and returns the
   new trash.Collector object

   trash.Start(trashlist) sends a trash list to the collector.

   trash.Dump() reports the collector's current trash list.

   trash.Close() shuts down the trash collector.
*/

type List struct {
	ExpirationTime int      `json:"expiration_time"`
	TrashBlocks    []string `json:"trash_blocks"`
}

type Collector struct {
	queue chan List
	dump  chan List
}

// New returns a new trash.Collector.  It launches a goroutine that
// waits for a list of blocks to trash.
//
func New() *Collector {
	c := Collector{
		make(chan List),
		make(chan List),
	}
	go c.listen()
	return &c
}

// Start sends a new trash list to the trash collector goroutine.  The
// collector will discard any old trash list and replace it with the
// new one.
func (c *Collector) Start(trashlist List) {
	c.queue <- trashlist
}

// Dump reports the contents of the current trash list.
func (c *Collector) Dump() List {
	return <-c.dump
}

// Close shuts down the trash collector.
//
func (c *Collector) Close() {
	close(c.queue)
}

// listen is run in a goroutine. It reads new pull lists from its
// input queue until the queue is closed.
func (c *Collector) listen() {
	var current List
	for {
		select {
		case newlist, ok := <-c.queue:
			if ok {
				current = newlist
			} else {
				// The input channel is closed; time to shut down
				close(c.dump)
				return
			}
		case c.dump <- current:
			// no-op
		}
	}
}
