// Framework for monitoring the Arvados container Queue, Locks container
// records, and runs goroutine callbacks which implement execution and
// monitoring of the containers.
package dispatch

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	Queued    = arvados.ContainerStateQueued
	Locked    = arvados.ContainerStateLocked
	Running   = arvados.ContainerStateRunning
	Complete  = arvados.ContainerStateComplete
	Cancelled = arvados.ContainerStateCancelled
)

// Dispatcher holds the state of the dispatcher
type Dispatcher struct {
	// The Arvados client
	Arv arvadosclient.ArvadosClient

	// When a new queued container appears and is either already owned by
	// this dispatcher or is successfully locked, the dispatcher will call
	// go RunContainer().  The RunContainer() goroutine gets a channel over
	// which it will receive updates to the container state.  The
	// RunContainer() goroutine should only assume status updates come when
	// the container record changes on the API server; if it needs to
	// monitor the job submission to the underlying slurm/grid engine/etc
	// queue it should spin up its own polling goroutines.  When the
	// channel is closed, that means the container is no longer being
	// handled by this dispatcher and the goroutine should terminate.  The
	// goroutine is responsible for draining the 'status' channel, failure
	// to do so may deadlock the dispatcher.
	RunContainer func(*Dispatcher, arvados.Container, chan arvados.Container)

	// Amount of time to wait between polling for updates.
	PollInterval time.Duration

	// Channel used to signal that RunDispatcher loop should exit.
	DoneProcessing chan struct{}

	mineMutex  sync.Mutex
	mineMap    map[string]chan arvados.Container
	Auth       arvados.APIClientAuthorization
	containers chan arvados.Container
}

// Goroutine-safely add/remove uuid to the set of "my" containers, i.e., ones
// for which this process is actively starting/monitoring.  Returns channel to
// be used to send container status updates.
func (dispatcher *Dispatcher) setMine(uuid string) chan arvados.Container {
	dispatcher.mineMutex.Lock()
	defer dispatcher.mineMutex.Unlock()
	if ch, ok := dispatcher.mineMap[uuid]; ok {
		return ch
	}

	ch := make(chan arvados.Container)
	dispatcher.mineMap[uuid] = ch
	return ch
}

// Release a container which is no longer being monitored.
func (dispatcher *Dispatcher) notMine(uuid string) {
	dispatcher.mineMutex.Lock()
	defer dispatcher.mineMutex.Unlock()
	if ch, ok := dispatcher.mineMap[uuid]; ok {
		close(ch)
		delete(dispatcher.mineMap, uuid)
	}
}

// checkMine returns true if there is a channel for updates associated
// with container c.  If update is true, also send the container record on
// the channel.
func (dispatcher *Dispatcher) checkMine(c arvados.Container, update bool) bool {
	dispatcher.mineMutex.Lock()
	defer dispatcher.mineMutex.Unlock()
	ch, ok := dispatcher.mineMap[c.UUID]
	if ok {
		if update {
			ch <- c
		}
		return true
	}
	return false
}

func (dispatcher *Dispatcher) getContainers(params arvadosclient.Dict, touched map[string]bool) {
	var containers arvados.ContainerList
	err := dispatcher.Arv.List("containers", params, &containers)
	if err != nil {
		log.Printf("Error getting list of containers: %q", err)
		return
	}

	if containers.ItemsAvailable > len(containers.Items) {
		// TODO: support paging
		log.Printf("Warning!  %d containers are available but only received %d, paged requests are not yet supported, some containers may be ignored.",
			containers.ItemsAvailable,
			len(containers.Items))
	}
	for _, container := range containers.Items {
		touched[container.UUID] = true
		dispatcher.containers <- container
	}
}

func (dispatcher *Dispatcher) pollContainers() {
	ticker := time.NewTicker(dispatcher.PollInterval)

	paramsQ := arvadosclient.Dict{
		"filters": [][]interface{}{{"state", "=", "Queued"}, {"priority", ">", "0"}},
		"order":   []string{"priority desc"},
		"limit":   "1000"}
	paramsP := arvadosclient.Dict{
		"filters": [][]interface{}{{"locked_by_uuid", "=", dispatcher.Auth.UUID}},
		"limit":   "1000"}

	for {
		select {
		case <-ticker.C:
			touched := make(map[string]bool)
			dispatcher.getContainers(paramsQ, touched)
			dispatcher.getContainers(paramsP, touched)
			dispatcher.mineMutex.Lock()
			var monitored []string
			for k := range dispatcher.mineMap {
				if _, ok := touched[k]; !ok {
					monitored = append(monitored, k)
				}
			}
			dispatcher.mineMutex.Unlock()
			if monitored != nil {
				dispatcher.getContainers(arvadosclient.Dict{
					"filters": [][]interface{}{{"uuid", "in", monitored}}}, touched)
			}
		case <-dispatcher.DoneProcessing:
			close(dispatcher.containers)
			ticker.Stop()
			return
		}
	}
}

func (dispatcher *Dispatcher) handleUpdate(container arvados.Container) {
	if container.State == Queued && dispatcher.checkMine(container, false) {
		// If we previously started the job, something failed, and it
		// was re-queued, this dispatcher might still be monitoring it.
		// Stop the existing monitor, then try to lock and run it
		// again.
		dispatcher.notMine(container.UUID)
	}

	if container.LockedByUUID != dispatcher.Auth.UUID && container.State != Queued {
		// If container is Complete, Cancelled, or Queued, LockedByUUID
		// will be nil.  If the container was formerly Locked, moved
		// back to Queued and then locked by another dispatcher,
		// LockedByUUID will be different.  In either case, we want
		// to stop monitoring it.
		log.Printf("Container %v now in state %q with locked_by_uuid %q", container.UUID, container.State, container.LockedByUUID)
		dispatcher.notMine(container.UUID)
		return
	}

	if dispatcher.checkMine(container, true) {
		// Already monitored, sent status update
		return
	}

	if container.State == Queued && container.Priority > 0 {
		// Try to take the lock
		if err := dispatcher.UpdateState(container.UUID, Locked); err != nil {
			return
		}
		container.State = Locked
	}

	if container.State == Locked || container.State == Running {
		// Not currently monitored but in Locked or Running state and
		// owned by this dispatcher, so start monitoring.
		go dispatcher.RunContainer(dispatcher, container, dispatcher.setMine(container.UUID))
	}
}

// UpdateState makes an API call to change the state of a container.
func (dispatcher *Dispatcher) UpdateState(uuid string, newState arvados.ContainerState) error {
	err := dispatcher.Arv.Update("containers", uuid,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": newState}},
		nil)
	if err != nil {
		log.Printf("Error updating container %s to state %q: %q", uuid, newState, err)
	}
	return err
}

// RunDispatcher runs the main loop of the dispatcher until receiving a message
// on the dispatcher.DoneProcessing channel.  It also installs a signal handler
// to terminate gracefully on SIGINT, SIGTERM or SIGQUIT.
func (dispatcher *Dispatcher) RunDispatcher() (err error) {
	err = dispatcher.Arv.Call("GET", "api_client_authorizations", "", "current", nil, &dispatcher.Auth)
	if err != nil {
		log.Printf("Error getting my token UUID: %v", err)
		return
	}

	dispatcher.mineMap = make(map[string]chan arvados.Container)
	dispatcher.containers = make(chan arvados.Container)

	// Graceful shutdown on signal
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func(sig <-chan os.Signal) {
		for sig := range sig {
			log.Printf("Caught signal: %v", sig)
			dispatcher.DoneProcessing <- struct{}{}
		}
	}(sigChan)

	defer close(sigChan)
	defer signal.Stop(sigChan)

	go dispatcher.pollContainers()
	for container := range dispatcher.containers {
		dispatcher.handleUpdate(container)
	}

	return nil
}
