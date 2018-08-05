// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// Dispatcher service for Crunch that runs containers on elastic cloud VMs

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
)

var version = "dev"

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%q", err)
	}
}

var (
	runningCmds      map[string]*exec.Cmd
	runningCmdsMutex sync.Mutex
	waitGroup        sync.WaitGroup
	crunchRunCommand *string
)

const (
	Booting  = iota
	Idle     = iota
	Busy     = iota
	Shutdown = iota
)

type Node struct {
	uuid         string
	instanceType string
	state        int
	allocated    string
	status       chan int
	ipaddr       string
	gone         bool
}

type NodeRequest struct {
	Container    arvados.Container
	instanceType string
	ready        chan *Node
}

type Scheduler struct {
	schedulerMutex sync.Mutex

	// node id to node
	nodes map[string]*Node

	// container to node
	containerToNode map[string]*Node

	// instance type to node
	typeToNodes map[string][]*Node

	requests []*NodeRequest
}

func (sch *Scheduler) setup() {
	sch.Dispatcher = &dispatch.Dispatcher{
		Arv:          arv,
		RunContainer: sch.runContainer,
		//PollPeriod:     time.Duration(disp.PollPeriod),
		//MinRetryPeriod: time.Duration(disp.MinRetryPeriod),
	}

	go sch.schedule()
}

func startFunc(container arvados.Container, cmd *exec.Cmd) error {
	return cmd.Start()
}

var startCmd = startFunc

func (sch *Scheduler) allocateNode(nr *NodeRequest) {
	for _, n := range sch.typeToNodes[nr.instanceType] {
		if n.allocated == "" && n.state == Idle {
			n.allocated = nr.Container.UUID
			containerToNode[nr.Container.UUID] = n
			return n
		}
	}
	return nil
}

func (sch *Scheduler) removeNode(newnode *Node) {
	delete(sch.nodes, newnode.uuid)
	ns := sch.typeToNodes[newnode.instancetype]
	for i, n := range ns {
		if n == newnode {
			ns[i] = ns[len(ns)-1]
			sch.typeToNodes[newnode.instancetype] = ns[0 : len(ns)-1]
			return
		}
	}
}

func (sch *Scheduler) createCloudNode(newnode *Node) {
	err = sch.driver.CreateNode()
	if err != nil {
		sch.removeNode(newnode)
	}
}

func (sch *Scheduler) deleteCloudNode(node *Node) {
	node.ipaddr = ""
	err = sch.driver.DeleteNode(node)
}

func (sch *Scheduler) nodeMonitor(node *Node) {
	for {
		if node.gone {
			sch.removeNode(node)
			break
		}

		if node.ipaddr == "" {
			continue
		}
		session := ssh(node.ipaddr)
		status := session.getStatus()
		node.allocated = status.allocated
		node.state = status.state
		node.lastStateChange = status.lastStateChange

		if node.lastStateChange > time.Duration(5*time.Minutes) && node.state == Idle {
			node.state = Shutdown
			sch.deleteCloudNode(node)
		}
	}
}

func (sch *Scheduler) cloudNodeList() {
	for {
		cloudNodes := sch.driver.CloudNodeList()
		seen := make(map[string]bool)
		for _, cl := range cloudNodes {
			uuid := cl.Tag["crunch-uuid"]
			instanceType := cl.Tag["crunch-instancetype"]
			noderecord, found := sch.nodes[uuid]
			seen[uuid] = true
			if !found {
				noderecord = Node{
					uuid:      uuid,
					state:     Booting,
					allocated: "",
					make(chan int),
					instanceType: instanceType}
			}
			if noderecord.ipaddr == "" {
				noderecord.ipaddr = cl.ipaddr
			}

			if !found {
				go sch.nodeMonitor(noderecord)
			}
		}
		for uuid, node := range sch.nodes {
			if seen[uuid] == false && node.state != Booting {
				if node.allocated != "" {
					arv.CancelContainer(node.allocated)
					node.gone = true
				}
			}
		}
	}
}

func (sch *Scheduler) schedule() {
	for {
		unallocated := make([]*NodeRequest, 0, len(sch.requests))

		bootingCounts := make(map[string]int)
		for t := range sch.typeToNodes {
			bootingCounts[t] = len(sch.typeToNodes[t])
		}

		for _, nr := range sch.requests {
			node := allocateNode(r)
			if node != nil {
				nr.ready <- node
				continue
			}
			unallocated = append(unallocated, nr)
			if bootingCounts[nr.instanceType] > 0 {
				bootingCounts[nr.instanceType] -= 1
				continue
			}

			newnode := Node{
				uuid:      "random uuid goes here",
				state:     Booting,
				allocated: "",
				make(chan int),
				instanceType: nr.instanceType}
			sch.nodes[newnode.uuid] = newnode
			sch.typeToNodes[newnode.instancetype] = append(sch.typeToNodes[newnode.instancetype], newnode)

			go sch.createCloudNode(newnode)
		}
		sch.requests = unallocated
	}
}

func (sch *Scheduler) cancelRequest(nr *NodeRequest) {
	close(nr.ready)
}

func (sch *Scheduler) requestNode(ctr arvados.Container, status <-chan arvados.Container) *Node {
	sch.schedulerMutex.Lock()

	if n, ok := sch.containerToNode[ctr.UUID]; ok {
		sch.schedulerMutex.Unlock()
		return n
	}

	nr := NodeRequest{Container: ctr, ready: make(chan *Node, 1)}

	sch.requests = append(sch.requests, nr)

	sch.schedulerMutex.Unlock()

	go func() {
		for {
			st <- status
			if st.State != "Locked" || st.Priority == 0 {
				sch.cancelRequest(nr)
				return
			}
		}
	}()

	return <-nr.ready
}

func (sch *Scheduler) releaseNode(n *Node) {
	sch.schedulerMutex.Lock()
	defer sch.schedulerMutex.Unlock()

	n.allocated = ""
	delete(containerToNode, ctr.UUID)
}

func (sch *Scheduler) runContainer(_ *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node := sch.requestNode(ctr, status)

	// node is allocated to us, but in a recovery scenario it may
	// already be running the container.
	if node.state == Idle {
		node.ssh.crunchRun(ctr)
	}

	// Assume node should now be busy
	for {
		st <- status
		if st.Priority == 0 {
			node.ssh.Signal(os.Interrupt)
		}
		if st.State == dispatch.Completed || st.State == dispatch.Cancelled {
			sch.releaseNode(node)
			return
		}
	}
}

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-cloud", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	crunchRunCommand = flags.String(
		"crunch-run-command",
		"/usr/bin/crunch-run",
		"Crunch command to run container")

	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Print version information if requested
	if *getVersion {
		fmt.Printf("crunch-dispatch-cloud %s\n", version)
		return nil
	}

	log.Printf("crunch-dispatch-cloud %s started", version)

	runningCmds = make(map[string]*exec.Cmd)

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("Error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		RunContainer: run,
		PollPeriod:   time.Duration(*pollInterval) * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = dispatcher.Run(ctx)
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-c
	log.Printf("Received %s, shutting down", sig)
	signal.Stop(c)

	cancel()

	runningCmdsMutex.Lock()
	// Finished dispatching; interrupt any crunch jobs that are still running
	for _, cmd := range runningCmds {
		cmd.Process.Signal(os.Interrupt)
	}
	runningCmdsMutex.Unlock()

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	return nil
}
