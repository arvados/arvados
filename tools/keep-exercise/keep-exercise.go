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
	"crypto/rand"
	"encoding/binary"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

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
)

func main() {
	flag.Parse()

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatal(err)
	}
	kc, err := keepclient.MakeKeepClient(&arv)
	if err != nil {
		log.Fatal(err)
	}
	kc.Want_replicas = *Replicas
	kc.Client.Timeout = 10 * time.Minute

	overrideServices(kc)

	nextBuf := make(chan []byte, *WriteThreads)
	nextLocator := make(chan string, *ReadThreads+*WriteThreads)

	go countBeans(nextLocator)
	for i := 0; i < *WriteThreads; i++ {
		go makeBufs(nextBuf, i)
		go doWrites(kc, nextBuf, nextLocator)
	}
	for i := 0; i < *ReadThreads; i++ {
		go doReads(kc, nextLocator)
	}
	<-make(chan struct{})
}

// Send 1234 to bytesInChan when we receive 1234 bytes from keepstore.
var bytesInChan = make(chan uint64)
var bytesOutChan = make(chan uint64)

// Send struct{}{} to errorsChan when an error happens.
var errorsChan = make(chan struct{})

func countBeans(nextLocator chan string) {
	t0 := time.Now()
	var tickChan <-chan time.Time
	if *StatsInterval > 0 {
		tickChan = time.NewTicker(*StatsInterval).C
	}
	var bytesIn uint64
	var bytesOut uint64
	var errors uint64
	for {
		select {
		case <-tickChan:
			elapsed := time.Since(t0)
			log.Printf("%v elapsed: read %v bytes (%.1f MiB/s), wrote %v bytes (%.1f MiB/s), errors %d",
				elapsed,
				bytesIn, (float64(bytesIn) / elapsed.Seconds() / 1048576),
				bytesOut, (float64(bytesOut) / elapsed.Seconds() / 1048576),
				errors,
			)
		case i := <-bytesInChan:
			bytesIn += i
		case o := <-bytesOutChan:
			bytesOut += o
		case <-errorsChan:
			errors++
		}
	}
}

func makeBufs(nextBuf chan []byte, threadID int) {
	buf := make([]byte, *BlockSize)
	if *VaryThread {
		binary.PutVarint(buf, int64(threadID))
	}
	for {
		if *VaryRequest {
			buf = make([]byte, *BlockSize)
			if _, err := io.ReadFull(rand.Reader, buf); err != nil {
				log.Fatal(err)
			}
		}
		nextBuf <- buf
	}
}

func doWrites(kc *keepclient.KeepClient, nextBuf chan []byte, nextLocator chan string) {
	for buf := range nextBuf {
		locator, _, err := kc.PutB(buf)
		if err != nil {
			log.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		bytesOutChan <- uint64(len(buf))
		for cap(nextLocator) > len(nextLocator)+*WriteThreads {
			// Give the readers something to do, unless
			// they have lots queued up already.
			nextLocator <- locator
		}
	}
}

func doReads(kc *keepclient.KeepClient, nextLocator chan string) {
	for locator := range nextLocator {
		rdr, size, url, err := kc.Get(locator)
		if err != nil {
			log.Print(err)
			errorsChan <- struct{}{}
			continue
		}
		n, err := io.Copy(ioutil.Discard, rdr)
		rdr.Close()
		if n != size || err != nil {
			log.Printf("Got %d bytes (expected %d) from %s: %v", n, size, url, err)
			errorsChan <- struct{}{}
			continue
			// Note we don't count the bytes received in
			// partial/corrupt responses: we are measuring
			// throughput, not resource consumption.
		}
		bytesInChan <- uint64(n)
	}
}

func overrideServices(kc *keepclient.KeepClient) {
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
			log.Fatalf("Service %q was not in list advertised by API %+q", *ServiceUUID, kc.GatewayRoots())
		}
	} else {
		return
	}
	kc.SetServiceRoots(roots, roots, roots)
}
