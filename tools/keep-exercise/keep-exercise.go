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
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mathRand "math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
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
	UseIndex      = flag.Bool("useIndex", false, "use the GetIndex call to get a list of blocks to read. Requires the SystemRoot token. Use this to rule out caching effects when reading.")
)

func createKeepClient(stderr *log.Logger) (kc *keepclient.KeepClient) {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		stderr.Fatal(err)
	}
	kc, err = keepclient.MakeKeepClient(arv)
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
	overrideServices(kc, stderr)
	return kc
}

func main() {
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keep-exercise %s\n", version)
		os.Exit(0)
	}

	stderr := log.New(os.Stderr, "", log.LstdFlags)

	if *ReadThreads > 0 && *WriteThreads == 0 && !*UseIndex {
		stderr.Fatal("At least one write thread is required if rthreads is non-zero and useIndex is not enabled")
	}

	if *ReadThreads == 0 && *WriteThreads == 0 {
		stderr.Fatal("Nothing to do!")
	}

	kc := createKeepClient(stderr)

	// When UseIndx is set, we need a KeepClient with SystemRoot powers to get
	// the block index from the Keepstore. We use the SystemRootToken from
	// the Arvados config.yml for that.
	var cluster *arvados.Cluster
	if *ReadThreads > 0 && *UseIndex {
		cluster = loadConfig(stderr)
		kc.Arvados.ApiToken = cluster.SystemRootToken
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

	csvHeader := "Timestamp,Elapsed,Read (bytes),Avg Read Speed (MiB/s),Peak Read Speed (MiB/s),Written (bytes),Avg Write Speed (MiB/s),Peak Write Speed (MiB/s),Errors,ReadThreads,WriteThreads,VaryRequest,VaryThread,BlockSize,Replicas,StatsInterval,ServiceURL,ServiceUUID,UseIndex,RunTime,Repeat"
	var summary string

	var nextBufs []chan []byte
	for i := 0; i < *WriteThreads; i++ {
		nextBuf := make(chan []byte, 1)
		nextBufs = append(nextBufs, nextBuf)
		go makeBufs(nextBuf, i, stderr)
	}

	for i := 0; i < *Repeat; i++ {
		if ctx.Err() == nil {
			summary = runExperiment(ctx, cluster, kc, nextBufs, summary, csvHeader, stderr)
			stderr.Printf("*************************** experiment %d complete ******************************\n", i)
			summary += fmt.Sprintf(",%d\n", i)
		}
	}
	if ctx.Err() == nil {
		stderr.Println("Summary:")
		stderr.Println()
		fmt.Println()
		fmt.Println(csvHeader + ",Experiment")
		fmt.Println(summary)
	}
}

func runExperiment(ctx context.Context, cluster *arvados.Cluster, kc *keepclient.KeepClient, nextBufs []chan []byte, summary string, csvHeader string, stderr *log.Logger) (newSummary string) {
	// Send 1234 to bytesInChan when we receive 1234 bytes from keepstore.
	var bytesInChan = make(chan uint64)
	var bytesOutChan = make(chan uint64)
	// Send struct{}{} to errorsChan when an error happens.
	var errorsChan = make(chan struct{})

	var nextLocator atomic.Value
	// when UseIndex is set, this channel is used instead of nextLocator
	var indexLocatorChan = make(chan string, 2)

	newSummary = summary

	// Start warmup
	ready := make(chan struct{})
	var warmup bool
	if *ReadThreads > 0 {
		warmup = true
		if !*UseIndex {
			stderr.Printf("Start warmup phase, waiting for 1 available block before reading starts\n")
		} else {
			stderr.Printf("Start warmup phase, waiting for block index before reading starts\n")
		}
	}
	if warmup && !*UseIndex {
		go func() {
			locator, _, err := kc.PutB(<-nextBufs[0])
			if err != nil {
				stderr.Print(err)
				errorsChan <- struct{}{}
			}
			nextLocator.Store(locator)
			stderr.Println("Warmup complete!")
			close(ready)
		}()
	} else if warmup && *UseIndex {
		// Get list of blocks to read
		go getIndexLocators(ctx, cluster, kc, indexLocatorChan, stderr)
		select {
		case <-ctx.Done():
			return
		case <-indexLocatorChan:
			stderr.Println("Warmup complete!")
			close(ready)
		}
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
		go doWrites(ctx, kc, nextBufs[i], &nextLocator, bytesOutChan, errorsChan, stderr)
	}
	if *UseIndex {
		for i := 0; i < *ReadThreads; i++ {
			go doIndexReads(ctx, kc, cluster, indexLocatorChan, bytesInChan, errorsChan, stderr)
		}
	} else {
		for i := 0; i < *ReadThreads; i++ {
			go doReads(ctx, kc, &nextLocator, bytesInChan, errorsChan, stderr)
		}
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
			line := fmt.Sprintf("%v,%v,%v,%.1f,%.1f,%v,%.1f,%.1f,%d,%d,%d,%t,%t,%d,%d,%s,%s,%s,%t,%s,%d",
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
				*UseIndex,
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

func doWrites(ctx context.Context, kc *keepclient.KeepClient, nextBuf <-chan []byte, nextLocator *atomic.Value, bytesOutChan chan<- uint64, errorsChan chan<- struct{}, stderr *log.Logger) {
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

func getIndexLocators(ctx context.Context, cluster *arvados.Cluster, kc *keepclient.KeepClient, indexLocatorChan chan<- string, stderr *log.Logger) {
	if ctx.Err() == nil {
		var locators []string
		for uuid := range kc.LocalRoots() {
			reader, err := kc.GetIndex(uuid, "")
			if err != nil {
				stderr.Fatalf("Error getting index: %s\n", err)
			}
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				locators = append(locators, strings.Split(scanner.Text(), " ")[0])
			}
		}
		stderr.Printf("Found %d locators\n", len(locators))
		if len(locators) < 1 {
			stderr.Fatal("Error: no locators found. The keepstores do not seem to contain any data. Remove the useIndex cli argument.")
		}

		mathRand.Seed(time.Now().UnixNano())
		mathRand.Shuffle(len(locators), func(i, j int) { locators[i], locators[j] = locators[j], locators[i] })

		for _, locator := range locators {
			// We need the Collections.BlobSigningKey to sign our block requests. This requires access to /etc/arvados/config.yml
			signedLocator := arvados.SignLocator(locator, kc.Arvados.ApiToken, time.Now().Local().Add(1*time.Hour), cluster.Collections.BlobSigningTTL.Duration(), []byte(cluster.Collections.BlobSigningKey))
			select {
			case <-ctx.Done():
				return
			case indexLocatorChan <- signedLocator:
			}
		}
		stderr.Fatal("Error: ran out of locators to read!")
	}
}

func loadConfig(stderr *log.Logger) (cluster *arvados.Cluster) {
	loader := config.NewLoader(os.Stdin, nil)
	loader.SkipLegacy = true

	cfg, err := loader.Load()
	if err != nil {
		stderr.Fatal(err)
	}
	cluster, err = cfg.GetCluster("")
	if err != nil {
		stderr.Fatal(err)
	}
	return
}

func doIndexReads(ctx context.Context, kc *keepclient.KeepClient, cluster *arvados.Cluster, indexLocatorChan <-chan string, bytesInChan chan<- uint64, errorsChan chan<- struct{}, stderr *log.Logger) {
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case locator := <-indexLocatorChan:
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
}

func doReads(ctx context.Context, kc *keepclient.KeepClient, nextLocator *atomic.Value, bytesInChan chan<- uint64, errorsChan chan<- struct{}, stderr *log.Logger) {
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
