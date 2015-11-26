/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	"errors"
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
	"os"
	"time"
)

var (
	logEventTypePrefix  string
	logFrequencySeconds int
	minutesBetweenRuns  int
	dryRun              bool
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
		"How many minutes we wait between data manager runs. 0 means run once and exit.")
	flag.BoolVar(&dryRun,
		"dry-run",
		false,
		"Perform a dry run. Log how many blocks would be deleted/moved, but do not issue any changes to keepstore.")
}

func main() {
	flag.Parse()

	if minutesBetweenRuns == 0 {
		arv, err := arvadosclient.MakeArvadosClient()
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger, fmt.Sprintf("Error making arvados client: %v", err))
		}
		err = singlerun(arv)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger, fmt.Sprintf("singlerun: %v", err))
		}
	} else {
		waitTime := time.Minute * time.Duration(minutesBetweenRuns)
		for {
			log.Println("Beginning Run")
			arv, err := arvadosclient.MakeArvadosClient()
			if err != nil {
				loggerutil.FatalWithMessage(arvLogger, fmt.Sprintf("Error making arvados client: %v", err))
			}
			err = singlerun(arv)
			if err != nil {
				log.Printf("singlerun: %v", err)
			}
			log.Printf("Sleeping for %d minutes", minutesBetweenRuns)
			time.Sleep(waitTime)
		}
	}
}

var arvLogger *logger.Logger

func singlerun(arv arvadosclient.ArvadosClient) error {
	var err error
	if isAdmin, err := util.UserIsAdmin(arv); err != nil {
		return errors.New("Error verifying admin token: " + err.Error())
	} else if !isAdmin {
		return errors.New("Current user is not an admin. Datamanager requires a privileged token.")
	}

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

	if readCollections.Err != nil {
		return readCollections.Err
	}

	err = summary.MaybeWriteData(arvLogger, readCollections, keepServerInfo)
	if err != nil {
		return err
	}

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

	// Log that we're finished. We force the recording, since go will
	// not wait for the write timer before exiting.
	if arvLogger != nil {
		defer arvLogger.FinalUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			summaryInfo := logger.GetOrCreateMap(p, "summary_info")
			summaryInfo["block_replication_counts"] = bucketCounts
			summaryInfo["replication_summary"] = replicationCounts
			p["summary_info"] = summaryInfo

			p["run_info"].(map[string]interface{})["finished_at"] = time.Now()
			p["run_info"].(map[string]interface{})["args"] = os.Args
		})
	}

	// If dry-run, do not issue any changes to keepstore
	if dryRun {
		log.Printf("Datamanager dry-run. Returning without issuing any keepstore updates.")
		return nil
	}

	// Not dry-run; issue changes to keepstore
	kc, err := keepclient.MakeKeepClient(&arv)
	if err != nil {
		return fmt.Errorf("Error setting up keep client %v", err.Error())
	}

	pullServers := summary.ComputePullServers(kc,
		&keepServerInfo,
		readCollections.BlockToDesiredReplication,
		replicationSummary.UnderReplicatedBlocks)

	pullLists := summary.BuildPullLists(pullServers)

	trashLists, trashErr := summary.BuildTrashLists(kc,
		&keepServerInfo,
		replicationSummary.KeepBlocksNotInCollections)

	err = summary.WritePullLists(arvLogger, pullLists)
	if err != nil {
		return err
	}

	if trashErr != nil {
		return err
	}
	keep.SendTrashLists(kc, trashLists)

	return nil
}

// BuildDataFetcher returns a data fetcher that fetches data from remote servers.
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

		var err error
		*keepServerInfo, err = keep.GetKeepServersAndSummarize(
			keep.GetKeepServersParams{
				Client: arv,
				Logger: arvLogger,
				Limit:  1000})

		if err != nil {
			return
		}

		*readCollections = <-collectionChannel
	}
}
