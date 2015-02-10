/* Datamanager-specific logging methods. */

package loggerutil

import (
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"log"
	"os"
	"runtime"
	"time"
)

// Useful to call at the begining of execution to log info about the
// current run.
func LogRunInfo(arvLogger *logger.Logger) {
	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			runInfo := make(map[string]interface{})
			runInfo["time_started"] = now
			runInfo["args"] = os.Args
			hostname, err := os.Hostname()
			if err != nil {
				runInfo["hostname_error"] = err.Error()
			} else {
				runInfo["hostname"] = hostname
			}
			runInfo["pid"] = os.Getpid()
			p["run_info"] = runInfo
		})
	}
}

// A LogMutator that records the current memory usage. This is most useful as a logger write hook.
//
// Assumes we already have a map named "run_info" in properties. LogRunInfo() can create such a map for you if you call it.
func LogMemoryAlloc(p map[string]interface{}, e map[string]interface{}) {
	runInfo := p["run_info"].(map[string]interface{})
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	runInfo["alloc_bytes_in_use"] = memStats.Alloc
}

func FatalWithMessage(arvLogger *logger.Logger, message string) {
	if arvLogger != nil {
		arvLogger.FinalUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			p["FATAL"] = message
			p["run_info"].(map[string]interface{})["time_finished"] = time.Now()
		})
	}

	log.Fatalf(message)
}
