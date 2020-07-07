// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Testing tool for Keep services.
//
// keepexercise helps measure throughput and test reliability under
// various usage patterns.
//
// By default, it reads and writes blocks containing 2^26 NUL
// bytes. This generates network traffic without consuming much disk
// space.
//
// For a more realistic test, enable -vary-request. Warning: this will
// fill your storage volumes with random data if you leave it running,
// which can cost you money or leave you with too little room for
// useful data.
//
package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

var version = "dev"

// Command line config knobs
var (
	BlockSize     = flag.Int("block-size", keepclient.BLOCKSIZE, "bytes per read/write op")
	ReadThreads   = flag.Int("rthreads", 1, "number of concurrent readers")
	WriteThreads  = flag.Int("wthreads", 1, "number of concurrent writers")
	VaryRequest   = flag.Bool("vary-request", false, "vary the data for each request: consumes disk space, exercises write behavior")
	VaryThread    = flag.Bool("vary-thread", false, "use -wthreads different data blocks")
	Replicas      = flag.Int("replicas", 1, "replication level for writing")
	StatsInterval = flag.Duration("stats-interval", time.Second, "time interval between IO stats reports, or 0 to disable")
	ServiceURL    = flag.String("url", "", "specify scheme://host of a single keep service to exercise (instead of using all advertised services like normal clients)")
	ServiceUUID   = flag.String("uuid", "", "specify UUID of a single advertised keep service to exercise")
	getVersion    = flag.Bool("version", false, "Print version information and exit.")
	RunTime       = flag.Duration("run-time", 0, "time to run (e.g. 60s), or 0 to run indefinitely (default)")
	Repeat        = flag.Int("repeat", 1, "number of times to repeat the experiment (default 1)")
)

// Send 1234 to bytesInChan when we receive 1234 bytes from keepstore.
var bytesInChan = make(chan uint64)
var bytesOutChan = make(chan uint64)

// Send struct{}{} to errorsChan when an error happens.
var errorsChan = make(chan struct{})

func main() {
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keep-exercise %s\n", version)
		os.Exit(0)
	}

	stderr := log.New(os.Stderr, "", log.LstdFlags)

	if *ReadThreads > 0 && *WriteThreads == 0 {
		stderr.Fatal("At least one write thread is required if rthreads is non-zero")
	}

	if *ReadThreads == 0 && *WriteThreads == 0 {
		stderr.Fatal("Nothing to do!")
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		stderr.Fatal(err)
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		stderr.Fatal(err)
	}
	kc.Want_replicas = *Replicas

	kc.HTTPClient = &http.Client{
		Timeout: 10 * time.Minute,
		// It's not safe to copy *http.DefaultTransport
		// because it has a mutex (which might be locked)
		// protecting a private map (which might not be nil).
		// So we build our own, using the Go 1.12 default
		// values.
		Transport: &http.Transport{
			TLSClientConfig: arvadosclient.MakeTLSConfig(arv.ApiInsecure),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Print("\r") // Suppress the ^C print
		cancel()
	}()

	overrideServices(kc, stderr)
	csvHeader := "Timestamp,Elapsed,Read (bytes),Avg Read Speed (MiB/s),Peak Read Speed (MiB/s),Written (bytes),Avg Write Speed (MiB/s),Peak Write Speed (MiB/s),Errors,ReadThreads,WriteThreads,VaryRequest,VaryThread,BlockSize,Replicas,StatsInterval,ServiceURL,ServiceUUID,RunTime,Repeat"
	var summary string

	for i := 0; i < *Repeat; i++ {
		if ctx.Err() == nil {
			summary = runExperiment(ctx, kc, summary, csvHeader, stderr)
			stderr.Printf("*************************** experiment %d complete ******************************\n", i)
			summary += fmt.Sprintf(",%d\n", i)
		}
	}
	stderr.Println("Summary:")
	stderr.Println()
	fmt.Println()
	fmt.Println(csvHeader + ",Experiment")
	fmt.Println(summary)
}

func runExperiment(ctx context.Context, kc *keepclient.KeepClient, summary string, csvHeader string, stderr *log.Logger) (newSummary string) {
	newSummary = summary
	var nextLocator atomic.Value

	// Start warmup
	ready := make(chan struct{})
	var warmup bool
	if *ReadThreads > 0 {
		warmup = true
		stderr.Printf("Start warmup phase, waiting for 1 available block before reading starts\n")
	}
	nextBuf := make(chan []byte, 1)
	go makeBufs(nextBuf, 0, stderr)
	if warmup {
		go func() {
			locator, _, err := kc.PutB(<-nextBuf)
			if err != nil {
				stderr.Print(err)
				errorsChan <- struct{}{}
			}
			nextLocator.Store(locator)
			stderr.Println("Warmup complete!")
			close(ready)
		}()
	} else {
		close(ready)
	}
	select {
	case <-ctx.Done():
		return
	case <-ready:
	}

	// Warmup complete
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(*RunTime))
	defer cancel()

	for i := 0; i < *WriteThreads; i++ {
		if i > 0 {
			// the makeBufs goroutine with index 0 was already started for the warmup phase, above
			nextBuf := make(chan []byte, 1)
			go makeBufs(nextBuf, i, stderr)
		}
		go doWrites(ctx, kc, nextBuf, &nextLocator, stderr)
	}
	for i := 0; i < *ReadThreads; i++ {
		go doReads(ctx, kc, &nextLocator, stderr)
	}

	t0 := time.Now()
	var tickChan <-chan time.Time
	if *StatsInterval > 0 {
		tickChan = time.NewTicker(*StatsInterval).C
	}
	var bytesIn uint64
	var bytesOut uint64
	var errors uint64
	var rateIn, rateOut float64
	var maxRateIn, maxRateOut float64
	var exit, printCsv bool
	csv := log.New(os.Stdout, "", 0)
	csv.Println()
	csv.Println(csvHeader)
	for {
		select {
		case <-ctx.Done():
			printCsv = true
			exit = true
		case <-tickChan:
			printCsv = true
		case i := <-bytesInChan:
			bytesIn += i
		case o := <-bytesOutChan:
			bytesOut += o
		case <-errorsChan:
			errors++
		}
		if printCsv {
			elapsed := time.Since(t0)
			rateIn = float64(bytesIn) / elapsed.Seconds() / 1048576
			if rateIn > maxRateIn {
				maxRateIn = rateIn
			}
			rateOut = float64(bytesOut) / elapsed.Seconds() / 1048576
			if rateOut > maxRateOut {
				maxRateOut = rateOut
			}
			line := fmt.Sprintf("%v,%v,%v,%.1f,%.1f,%v,%.1f,%.1f,%d,%d,%d,%t,%t,%d,%d,%s,%s,%s,%s,%d",
				time.Now().Format("2006/01/02 15:04:05"),
				elapsed,
				bytesIn, rateIn, maxRateIn,
				bytesOut, rateOut, maxRateOut,
				errors,
				*ReadThreads,
				*WriteThreads,
				*VaryRequest,
				*VaryThread,
				*BlockSize,
				*Replicas,
				*StatsInterval,
				*ServiceURL,
				*ServiceUUID,
				*RunTime,
				*Repeat,
			)
			csv.Println(line)
			if exit {
				newSummary += line
				return
			}
			printCsv = false
		}
	}
	return
}

func makeBufs(nextBuf chan<- []byte, threadID int, stderr *log.Logger) {
	buf := make([]byte, *BlockSize)
	if *VaryThread {
		binary.PutVarint(buf, int64(threadID))
	}
	randSize := 524288
	if randSize > *BlockSize {
		randSize = *BlockSize
	}
	for {
		if *VaryRequest {
			rnd := make([]byte, randSize)
			if _, err := io.ReadFull(rand.Reader, rnd); err != nil {
				stderr.Fatal(err)
			}
			buf = append(rnd, buf[randSize:]...)
		}
		nextBuf <- buf
	}
}

func doWrites(ctx context.Context, kc *keepclient.KeepClient, nextBuf <-chan []byte, nextLocator *atomic.Value, stderr *log.Logger) {
	for ctx.Err() == nil {
		buf := <-nextBuf
		locator, _, err := kc.PutB(buf)
		if err != nil {
			stderr.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		bytesOutChan <- uint64(len(buf))
		nextLocator.Store(locator)
	}
}

func doReads(ctx context.Context, kc *keepclient.KeepClient, nextLocator *atomic.Value, stderr *log.Logger) {
	var locator string
	for ctx.Err() == nil {
		locator = nextLocator.Load().(string)
		rdr, size, url, err := kc.Get(locator)
		if err != nil {
			stderr.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		n, err := io.Copy(ioutil.Discard, rdr)
		rdr.Close()
		if n != size || err != nil {
			stderr.Printf("Got %d bytes (expected %d) from %s: %v", n, size, url, err)
			errorsChan <- struct{}{}
			continue
			// Note we don't count the bytes received in
			// partial/corrupt responses: we are measuring
			// throughput, not resource consumption.
		}
		bytesInChan <- uint64(n)
	}
}

func overrideServices(kc *keepclient.KeepClient, stderr *log.Logger) {
	roots := make(map[string]string)
	if *ServiceURL != "" {
		roots["zzzzz-bi6l4-000000000000000"] = *ServiceURL
	} else if *ServiceUUID != "" {
		for uuid, url := range kc.GatewayRoots() {
			if uuid == *ServiceUUID {
				roots[uuid] = url
				break
			}
		}
		if len(roots) == 0 {
			stderr.Fatalf("Service %q was not in list advertised by API %+q", *ServiceUUID, kc.GatewayRoots())
		}
	} else {
		return
	}
	kc.SetServiceRoots(roots, roots, roots)
}
