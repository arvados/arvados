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
			runInfo := logger.GetOrCreateMap(p, "run_info")
			runInfo["started_at"] = now
			runInfo["args"] = os.Args
			hostname, err := os.Hostname()
			if err != nil {
				runInfo["hostname_error"] = err.Error()
			} else {
				runInfo["hostname"] = hostname
			}
			runInfo["pid"] = os.Getpid()
		})
	}
}

// A LogMutator that records the current memory usage. This is most useful as a logger write hook.
func LogMemoryAlloc(p map[string]interface{}, e map[string]interface{}) {
	runInfo := logger.GetOrCreateMap(p, "run_info")
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	runInfo["memory_bytes_in_use"] = memStats.Alloc
	runInfo["memory_bytes_reserved"] = memStats.Sys
}

func FatalWithMessage(arvLogger *logger.Logger, message string) {
	if arvLogger != nil {
		arvLogger.FinalUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			p["FATAL"] = message
			runInfo := logger.GetOrCreateMap(p, "run_info")
			runInfo["finished_at"] = time.Now()
		})
	}

	log.Fatalf(message)
}
