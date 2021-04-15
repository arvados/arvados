// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
)

// RefreshServiceDiscovery clears the Keep service discovery cache.
func RefreshServiceDiscovery() {
	var wg sync.WaitGroup
	defer wg.Wait()
	svcListCacheMtx.Lock()
	defer svcListCacheMtx.Unlock()
	for _, ent := range svcListCache {
		wg.Add(1)
		clear := ent.clear
		go func() {
			clear <- struct{}{}
			wg.Done()
		}()
	}
}

// RefreshServiceDiscoveryOnSIGHUP installs a signal handler that calls
// RefreshServiceDiscovery when SIGHUP is received.
func RefreshServiceDiscoveryOnSIGHUP() {
	svcListCacheMtx.Lock()
	defer svcListCacheMtx.Unlock()
	if svcListCacheSignal != nil {
		return
	}
	svcListCacheSignal = make(chan os.Signal, 1)
	signal.Notify(svcListCacheSignal, syscall.SIGHUP)
	go func() {
		for range svcListCacheSignal {
			RefreshServiceDiscovery()
		}
	}()
}

var (
	svcListCache       = map[string]cachedSvcList{}
	svcListCacheSignal chan os.Signal
	svcListCacheMtx    sync.Mutex
)

type cachedSvcList struct {
	arv    *arvadosclient.ArvadosClient
	latest chan svcList
	clear  chan struct{}
}

// Check for new services list every few minutes. Send the latest list
// to the "latest" channel as needed.
func (ent *cachedSvcList) poll() {
	wakeup := make(chan struct{})

	replace := make(chan svcList)
	go func() {
		wakeup <- struct{}{}
		current := <-replace
		for {
			select {
			case <-ent.clear:
				wakeup <- struct{}{}
				// Wait here for the next success, in
				// order to avoid returning stale
				// results on the "latest" channel.
				current = <-replace
			case current = <-replace:
			case ent.latest <- current:
			}
		}
	}()

	okDelay := 5 * time.Minute
	errDelay := 3 * time.Second
	timer := time.NewTimer(okDelay)
	for {
		select {
		case <-timer.C:
		case <-wakeup:
			if !timer.Stop() {
				// Lost race stopping timer; skip extra firing
				<-timer.C
			}
		}
		var next svcList
		err := ent.arv.Call("GET", "keep_services", "", "accessible", nil, &next)
		if err != nil {
			log.Printf("WARNING: Error retrieving services list: %v (retrying in %v)", err, errDelay)
			timer.Reset(errDelay)
			continue
		}
		replace <- next
		timer.Reset(okDelay)
	}
}

// discoverServices gets the list of available keep services from
// the API server.
//
// If a list of services is provided in the arvadosclient (e.g., from
// an environment variable or local config), that list is used
// instead.
//
// If an API call is made, the result is cached for 5 minutes or until
// ClearCache() is called, and during this interval it is reused by
// other KeepClients that use the same API server host.
func (kc *KeepClient) discoverServices() error {
	if kc.disableDiscovery {
		return nil
	}

	if kc.Arvados.KeepServiceURIs != nil {
		kc.disableDiscovery = true
		kc.foundNonDiskSvc = true
		kc.replicasPerService = 0
		roots := make(map[string]string)
		for i, uri := range kc.Arvados.KeepServiceURIs {
			roots[fmt.Sprintf("00000-bi6l4-%015d", i)] = uri
		}
		kc.setServiceRoots(roots, roots, roots)
		return nil
	}

	if kc.Arvados.ApiServer == "" {
		return fmt.Errorf("Arvados client is not configured (target API host is not set). Maybe env var ARVADOS_API_HOST should be set first?")
	}

	svcListCacheMtx.Lock()
	cacheEnt, ok := svcListCache[kc.Arvados.ApiServer]
	if !ok {
		arv := *kc.Arvados
		cacheEnt = cachedSvcList{
			latest: make(chan svcList),
			clear:  make(chan struct{}),
			arv:    &arv,
		}
		go cacheEnt.poll()
		svcListCache[kc.Arvados.ApiServer] = cacheEnt
	}
	svcListCacheMtx.Unlock()

	select {
	case <-time.After(time.Minute):
		return errors.New("timed out while getting initial list of keep services")
	case sl := <-cacheEnt.latest:
		return kc.loadKeepServers(sl)
	}
}

func (kc *KeepClient) RefreshServiceDiscovery() {
	svcListCacheMtx.Lock()
	ent, ok := svcListCache[kc.Arvados.ApiServer]
	svcListCacheMtx.Unlock()
	if !ok || kc.Arvados.KeepServiceURIs != nil || kc.disableDiscovery {
		return
	}
	ent.clear <- struct{}{}
}

// LoadKeepServicesFromJSON gets list of available keep services from
// given JSON and disables automatic service discovery.
func (kc *KeepClient) LoadKeepServicesFromJSON(services string) error {
	kc.disableDiscovery = true

	var list svcList
	dec := json.NewDecoder(strings.NewReader(services))
	if err := dec.Decode(&list); err != nil {
		return err
	}

	return kc.loadKeepServers(list)
}

func (kc *KeepClient) loadKeepServers(list svcList) error {
	listed := make(map[string]bool)
	localRoots := make(map[string]string)
	gatewayRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	// replicasPerService is 1 for disks; unknown or unlimited otherwise
	kc.replicasPerService = 1

	for _, service := range list.Items {
		scheme := "http"
		if service.SSL {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s:%d", scheme, service.Hostname, service.Port)

		// Skip duplicates
		if listed[url] {
			continue
		}
		listed[url] = true

		localRoots[service.Uuid] = url
		if service.ReadOnly == false {
			writableLocalRoots[service.Uuid] = url
			if service.SvcType != "disk" {
				kc.replicasPerService = 0
			}
		}

		if service.SvcType != "disk" {
			kc.foundNonDiskSvc = true
		}

		// Gateway services are only used when specified by
		// UUID, so there's nothing to gain by filtering them
		// by service type. Including all accessible services
		// (gateway and otherwise) merely accommodates more
		// service configurations.
		gatewayRoots[service.Uuid] = url
	}

	kc.setServiceRoots(localRoots, writableLocalRoots, gatewayRoots)
	return nil
}
