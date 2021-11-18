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

	"git.arvados.org/arvados.git/lib/cmd"
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
	UseIndex      = flag.Bool("use-index", false, "use the GetIndex call to get a list of blocks to read. Requires the SystemRoot token. Use this to rule out caching effects when reading.")
)

func createKeepClient(lgr *log.Logger) (kc *keepclient.KeepClient) {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		lgr.Fatal(err)
	}
	kc, err = keepclient.MakeKeepClient(arv)
	if err != nil {
		lgr.Fatal(err)
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
	overrideServices(kc, lgr)
	return kc
}

func main() {
	if ok, code := cmd.ParseFlags(flag.CommandLine, os.Args[0], os.Args[1:], "", os.Stderr); !ok {
		os.Exit(code)
	} else if *getVersion {
		fmt.Printf("%s %s\n", os.Args[0], version)
		return
	}

	lgr := log.New(os.Stderr, "", log.LstdFlags)

	if *ReadThreads > 0 && *WriteThreads == 0 && !*UseIndex {
		lgr.Fatal("At least one write thread is required if rthreads is non-zero and -use-index is not enabled")
	}

	if *ReadThreads == 0 && *WriteThreads == 0 {
		lgr.Fatal("Nothing to do!")
	}

	kc := createKeepClient(lgr)

	// When UseIndex is set, we need a KeepClient with SystemRoot powers to get
	// the block index from the Keepstore. We use the SystemRootToken from
	// the Arvados config.yml for that.
	var cluster *arvados.Cluster
	if *ReadThreads > 0 && *UseIndex {
		cluster = loadConfig(lgr)
		kc.Arvados.ApiToken = cluster.SystemRootToken
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		// FIXME
		//fmt.Print("\r") // Suppress the ^C print
		cancel()
	}()

	csvHeader := "Timestamp,Elapsed,Read (bytes),Avg Read Speed (MiB/s),Peak Read Speed (MiB/s),Written (bytes),Avg Write Speed (MiB/s),Peak Write Speed (MiB/s),Errors,ReadThreads,WriteThreads,VaryRequest,VaryThread,BlockSize,Replicas,StatsInterval,ServiceURL,ServiceUUID,UseIndex,RunTime,Repeat"
	var summary string

	var nextBufs []chan []byte
	for i := 0; i < *WriteThreads; i++ {
		nextBuf := make(chan []byte, 1)
		nextBufs = append(nextBufs, nextBuf)
		go makeBufs(nextBuf, i, lgr)
	}

	for i := 0; i < *Repeat && ctx.Err() == nil; i++ {
		summary = runExperiment(ctx, cluster, kc, nextBufs, summary, csvHeader, lgr)
		lgr.Printf("*************************** experiment %d complete ******************************\n", i)
		summary += fmt.Sprintf(",%d\n", i)
	}

	lgr.Println("Summary:")
	lgr.Println()
	fmt.Println()
	fmt.Println(csvHeader + ",Experiment")
	fmt.Println(summary)
}

func runExperiment(ctx context.Context, cluster *arvados.Cluster, kc *keepclient.KeepClient, nextBufs []chan []byte, summary string, csvHeader string, lgr *log.Logger) (newSummary string) {
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
			lgr.Printf("Start warmup phase, waiting for 1 available block before reading starts\n")
		} else {
			lgr.Printf("Start warmup phase, waiting for block index before reading starts\n")
		}
	}
	if warmup && !*UseIndex {
		go func() {
			locator, _, err := kc.PutB(<-nextBufs[0])
			if err != nil {
				lgr.Print(err)
				errorsChan <- struct{}{}
			}
			nextLocator.Store(locator)
			lgr.Println("Warmup complete!")
			close(ready)
		}()
	} else if warmup && *UseIndex {
		// Get list of blocks to read
		go getIndexLocators(ctx, cluster, kc, indexLocatorChan, lgr)
		select {
		case <-ctx.Done():
			return
		case <-indexLocatorChan:
			lgr.Println("Warmup complete!")
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
		go doWrites(ctx, kc, nextBufs[i], &nextLocator, bytesOutChan, errorsChan, lgr)
	}
	if *UseIndex {
		for i := 0; i < *ReadThreads; i++ {
			go doReads(ctx, kc, nil, indexLocatorChan, bytesInChan, errorsChan, lgr)
		}
	} else {
		for i := 0; i < *ReadThreads; i++ {
			go doReads(ctx, kc, &nextLocator, nil, bytesInChan, errorsChan, lgr)
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

func makeBufs(nextBuf chan<- []byte, threadID int, lgr *log.Logger) {
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
				lgr.Fatal(err)
			}
			buf = append(rnd, buf[randSize:]...)
		}
		nextBuf <- buf
	}
}

func doWrites(ctx context.Context, kc *keepclient.KeepClient, nextBuf <-chan []byte, nextLocator *atomic.Value, bytesOutChan chan<- uint64, errorsChan chan<- struct{}, lgr *log.Logger) {
	for ctx.Err() == nil {
		//lgr.Printf("%s nextbuf %s, waiting for nextBuf\n",nextBuf,time.Now())
		buf := <-nextBuf
		//lgr.Printf("%s nextbuf %s, done waiting for nextBuf\n",nextBuf,time.Now())
		locator, _, err := kc.PutB(buf)
		if err != nil {
			lgr.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		bytesOutChan <- uint64(len(buf))
		nextLocator.Store(locator)
	}
}

func getIndexLocators(ctx context.Context, cluster *arvados.Cluster, kc *keepclient.KeepClient, indexLocatorChan chan<- string, lgr *log.Logger) {
	if ctx.Err() != nil {
		return
	}
	locatorsMap := make(map[string]bool)
	var locators []string
	var count int64
	for uuid := range kc.LocalRoots() {
		reader, err := kc.GetIndex(uuid, "")
		if err != nil {
			lgr.Fatalf("Error getting index: %s\n", err)
		}
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			locatorsMap[strings.Split(scanner.Text(), " ")[0]] = true
			count++
		}
	}
	for l := range locatorsMap {
		locators = append(locators, l)
	}
	lgr.Printf("Found %d locators\n", count)
	lgr.Printf("Found %d locators (deduplicated)\n", len(locators))
	if len(locators) < 1 {
		lgr.Fatal("Error: no locators found. The keepstores do not seem to contain any data. Remove the -use-index cli argument.")
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
	lgr.Fatal("Error: ran out of locators to read!")
}

func loadConfig(lgr *log.Logger) (cluster *arvados.Cluster) {
	loader := config.NewLoader(os.Stdin, nil)
	loader.SkipLegacy = true

	cfg, err := loader.Load()
	if err != nil {
		lgr.Fatal(err)
	}
	cluster, err = cfg.GetCluster("")
	if err != nil {
		lgr.Fatal(err)
	}
	return
}

func doReads(ctx context.Context, kc *keepclient.KeepClient, nextLocator *atomic.Value, indexLocatorChan <-chan string, bytesInChan chan<- uint64, errorsChan chan<- struct{}, lgr *log.Logger) {
	for ctx.Err() == nil {
		var locator string
		if indexLocatorChan != nil {
			select {
			case <-ctx.Done():
				return
			case locator = <-indexLocatorChan:
			}
		} else {
			locator = nextLocator.Load().(string)
		}
		rdr, size, url, err := kc.Get(locator)
		if err != nil {
			lgr.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		n, err := io.Copy(ioutil.Discard, rdr)
		rdr.Close()
		if n != size || err != nil {
			lgr.Printf("Got %d bytes (expected %d) from %s: %v", n, size, url, err)
			errorsChan <- struct{}{}
			continue
			// Note we don't count the bytes received in
			// partial/corrupt responses: we are measuring
			// throughput, not resource consumption.
		}
		bytesInChan <- uint64(n)
	}
}

func overrideServices(kc *keepclient.KeepClient, lgr *log.Logger) {
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
			lgr.Fatalf("Service %q was not in list advertised by API %+q", *ServiceUUID, kc.GatewayRoots())
		}
	} else {
		return
	}
	kc.SetServiceRoots(roots, roots, roots)
}
