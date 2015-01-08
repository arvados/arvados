/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"log"
	"os"
	"time"
)

var (
	logEventType string
	logFrequencySeconds int
)

func init() {
	flag.StringVar(&logEventType, 
		"log-event-type",
		"experimental-data-manager-report",
		"event_type to use in our arvados log entries. Set to empty to turn off logging")
	flag.IntVar(&logFrequencySeconds, 
		"log-frequency-seconds",
		20,
		"How frequently we'll write log entries in seconds.")
}

func main() {
	flag.Parse()

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	if is_admin, err := util.UserIsAdmin(arv); err != nil {
		log.Fatalf("Error querying current arvados user %s", err.Error())
	} else if !is_admin {
		log.Fatalf("Current user is not an admin. Datamanager can only be run by admins.")
	}

	var arvLogger *logger.Logger
	if logEventType != "" {
		arvLogger = logger.NewLogger(logger.LoggerParams{Client: arv,
			EventType: logEventType,
			MinimumWriteInterval: time.Second * time.Duration(logFrequencySeconds)})
	}

	if arvLogger != nil {
		properties, _ := arvLogger.Edit()
		runInfo := make(map[string]interface{})
		runInfo["start_time"] = time.Now()
		runInfo["args"] = os.Args
		hostname, err := os.Hostname()
		if err != nil {
			runInfo["hostname_error"] = err.Error()
		} else {
			runInfo["hostname"] = hostname
		}
		runInfo["pid"] = os.Getpid()
		properties["run_info"] = runInfo
		arvLogger.Record()
	}

	// TODO(misha): Read Collections and Keep Contents concurrently as goroutines.
	// This requires waiting on them to finish before you let main() exit.

	RunCollections(collection.GetCollectionsParams{
		Client: arv, Logger: arvLogger, BatchSize: 500})

	RunKeep(keep.GetKeepServersParams{Client: arv, Limit: 1000})
}

func RunCollections(params collection.GetCollectionsParams) {
	readCollections := collection.GetCollections(params)

	UserUsage := ComputeSizeOfOwnedCollections(readCollections)
	log.Printf("Uuid to Size used: %v", UserUsage)

	// TODO(misha): Add a "readonly" flag. If we're in readonly mode,
	// lots of behaviors can become warnings (and obviously we can't
	// write anything).
	// if !readCollections.ReadAllCollections {
	// 	log.Fatalf("Did not read all collections")
	// }

	log.Printf("Read and processed %d collections",
		len(readCollections.UuidToCollection))
}

func RunKeep(params keep.GetKeepServersParams) {
	readServers := keep.GetKeepServers(params)

	log.Printf("Returned %d keep disks", len(readServers.ServerToContents))

	blockReplicationCounts := make(map[int]int)
	for _, infos := range readServers.BlockToServers {
		replication := len(infos)
		blockReplicationCounts[replication] += 1
	}

	log.Printf("Replication level distribution: %v", blockReplicationCounts)
}

func ComputeSizeOfOwnedCollections(readCollections collection.ReadCollections) (
	results map[string]int) {
	results = make(map[string]int)
	for _, coll := range readCollections.UuidToCollection {
		results[coll.OwnerUuid] = results[coll.OwnerUuid] + coll.TotalSize
	}
	return
}
