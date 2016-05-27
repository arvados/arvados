package dispatch

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Constants for container states
const (
	Queued    = "Queued"
	Locked    = "Locked"
	Running   = "Running"
	Complete  = "Complete"
	Cancelled = "Cancelled"
)

type apiClientAuthorization struct {
	UUID     string `json:"uuid"`
	APIToken string `json:"api_token"`
}

type apiClientAuthorizationList struct {
	Items []apiClientAuthorization `json:"items"`
}

// Container data
type Container struct {
	UUID               string           `json:"uuid"`
	State              string           `json:"state"`
	Priority           int              `json:"priority"`
	RuntimeConstraints map[string]int64 `json:"runtime_constraints"`
	LockedByUUID       string           `json:"locked_by_uuid"`
}

// ContainerList is a list of the containers from api
type ContainerList struct {
	Items          []Container `json:"items"`
	ItemsAvailable int         `json:"items_available"`
}

// Dispatcher holds the state of the dispatcher
type Dispatcher struct {
	Arv            arvadosclient.ArvadosClient
	RunContainer   func(*Dispatcher, Container, chan Container)
	PollInterval   time.Duration
	DoneProcessing chan struct{}

	mineMutex  sync.Mutex
	mineMap    map[string]chan Container
	Auth       apiClientAuthorization
	containers chan Container
}

// Goroutine-safely add/remove uuid to the set of "my" containers, i.e., ones
// for which this process is actively starting/monitoring.  Returns channel to
// be used to send container status updates.
func (dispatcher *Dispatcher) setMine(uuid string) chan Container {
	dispatcher.mineMutex.Lock()
	defer dispatcher.mineMutex.Unlock()
	if ch, ok := dispatcher.mineMap[uuid]; ok {
		return ch
	}

	ch := make(chan Container)
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

// Check if there is a channel for updates associated with this container.  If
// so send the container record on the channel and return true, if not return
// false.
func (dispatcher *Dispatcher) updateMine(c Container) bool {
	dispatcher.mineMutex.Lock()
	defer dispatcher.mineMutex.Unlock()
	ch, ok := dispatcher.mineMap[c.UUID]
	if ok {
		ch <- c
		return true
	}
	return false
}

func (dispatcher *Dispatcher) getContainers(params arvadosclient.Dict, touched map[string]bool) {
	var containers ContainerList
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

func (dispatcher *Dispatcher) handleUpdate(container Container) {
	if container.LockedByUUID != dispatcher.Auth.UUID && container.State != Queued {
		// If container is Complete, Cancelled, or Queued, LockedByUUID
		// will be nil.  If the container was formally Locked, moved
		// back to Queued and then locked by another dispatcher,
		// LockedByUUID will be different.  In either case, we want
		// to stop monitoring it.
		log.Printf("Container %v now in state %v with locked_by_uuid %v", container.UUID, container.State, container.LockedByUUID)
		dispatcher.notMine(container.UUID)
		return
	}

	if dispatcher.updateMine(container) {
		// Already monitored, sent status update
		return
	}

	if container.State == Queued {
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
func (dispatcher *Dispatcher) UpdateState(uuid, newState string) error {
	err := dispatcher.Arv.Update("containers", uuid,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": newState}},
		nil)
	if err != nil {
		log.Printf("Error updating container %s to '%s' state: %q", uuid, newState, err)
	}
	return err
}

// RunDispatcher runs the main loop of the dispatcher until receiving a message
// on the dispatcher.DoneProcessing channel.  It also installs a signal handler
// to terminate gracefully on SIGINT, SIGTERM or SIGQUIT.
//
// When a new queued container appears and is successfully locked, the
// dispatcher will call RunContainer() followed by MonitorContainer().  If a
// container appears that is Locked or Running but not known to the dispatcher,
// it will only call monitorContainer().  The monitorContainer() callback is
// passed a channel over which it will receive updates to the container state.
// The callback is responsible for draining the channel, if it fails to do so
// it will deadlock the dispatcher.
func (dispatcher *Dispatcher) RunDispatcher() (err error) {
	err = dispatcher.Arv.Call("GET", "api_client_authorizations", "", "current", nil, &dispatcher.Auth)
	if err != nil {
		log.Printf("Error getting my token UUID: %v", err)
		return
	}

	dispatcher.mineMap = make(map[string]chan Container)
	dispatcher.containers = make(chan Container)

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
