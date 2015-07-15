/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"git.curoverse.com/arvados.git/services/datamanager/summary"
	"log"
	"time"
)

var (
	logEventTypePrefix  string
	logFrequencySeconds int
	minutesBetweenRuns  int
)

func init() {
	flag.StringVar(&logEventTypePrefix,
		"log-event-type-prefix",
		"experimental-data-manager",
		"Prefix to use in the event_type of our arvados log entries. Set to empty to turn off logging")
	flag.IntVar(&logFrequencySeconds,
		"log-frequency-seconds",
		20,
		"How frequently we'll write log entries in seconds.")
	flag.IntVar(&minutesBetweenRuns,
		"minutes-between-runs",
		0,
		"How many minutes we wait betwen data manager runs. 0 means run once and exit.")
}

func main() {
	flag.Parse()
	if minutesBetweenRuns == 0 {
		singlerun()
	} else {
		waitTime := time.Minute * time.Duration(minutesBetweenRuns)
		for {
			log.Println("Beginning Run")
			singlerun()
			log.Printf("Sleeping for %d minutes", minutesBetweenRuns)
			time.Sleep(waitTime)
		}
	}
}

func singlerun() {
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
	if logEventTypePrefix != "" {
		arvLogger = logger.NewLogger(logger.LoggerParams{
			Client:          arv,
			EventTypePrefix: logEventTypePrefix,
			WriteInterval:   time.Second * time.Duration(logFrequencySeconds)})
	}

	loggerutil.LogRunInfo(arvLogger)
	if arvLogger != nil {
		arvLogger.AddWriteHook(loggerutil.LogMemoryAlloc)
	}

	var (
		dataFetcher     summary.DataFetcher
		readCollections collection.ReadCollections
		keepServerInfo  keep.ReadServers
	)

	if summary.ShouldReadData() {
		dataFetcher = summary.ReadData
	} else {
		dataFetcher = BuildDataFetcher(arv)
	}

	dataFetcher(arvLogger, &readCollections, &keepServerInfo)

	summary.MaybeWriteData(arvLogger, readCollections, keepServerInfo)

	buckets := summary.BucketReplication(readCollections, keepServerInfo)
	bucketCounts := buckets.Counts()

	replicationSummary := buckets.SummarizeBuckets(readCollections)
	replicationCounts := replicationSummary.ComputeCounts()

	log.Printf("Blocks In Collections: %d, "+
		"\nBlocks In Keep: %d.",
		len(readCollections.BlockToDesiredReplication),
		len(keepServerInfo.BlockToServers))
	log.Println(replicationCounts.PrettyPrint())

	log.Printf("Blocks Histogram:")
	for _, rlbss := range bucketCounts {
		log.Printf("%+v: %10d",
			rlbss.Levels,
			rlbss.Count)
	}

	kc, err := keepclient.MakeKeepClient(&arv)
	if err != nil {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Error setting up keep client %s", err.Error()))
	}

	pullServers := summary.ComputePullServers(kc,
		&keepServerInfo,
		readCollections.BlockToDesiredReplication,
		replicationSummary.UnderReplicatedBlocks)

	pullLists := summary.BuildPullLists(pullServers)
	trashLists := summary.BuildTrashLists(kc,
		&keepServerInfo,
		replicationSummary.KeepBlocksNotInCollections)

	summary.WritePullLists(arvLogger, pullLists)

	summary.WriteTrashLists(arvLogger, trashLists)

	// Log that we're finished. We force the recording, since go will
	// not wait for the write timer before exiting.
	if arvLogger != nil {
		arvLogger.FinalUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			summaryInfo := logger.GetOrCreateMap(p, "summary_info")
			summaryInfo["block_replication_counts"] = bucketCounts
			summaryInfo["replication_summary"] = replicationCounts
			p["summary_info"] = summaryInfo

			p["run_info"].(map[string]interface{})["finished_at"] = time.Now()
		})
	}
}

// Returns a data fetcher that fetches data from remote servers.
func BuildDataFetcher(arv arvadosclient.ArvadosClient) summary.DataFetcher {
	return func(arvLogger *logger.Logger,
		readCollections *collection.ReadCollections,
		keepServerInfo *keep.ReadServers) {
		collectionChannel := make(chan collection.ReadCollections)

		go func() {
			collectionChannel <- collection.GetCollectionsAndSummarize(
				collection.GetCollectionsParams{
					Client:    arv,
					Logger:    arvLogger,
					BatchSize: 50})
		}()

		*keepServerInfo = keep.GetKeepServersAndSummarize(
			keep.GetKeepServersParams{
				Client: arv,
				Logger: arvLogger,
				Limit:  1000})

		*readCollections = <-collectionChannel
	}
}
