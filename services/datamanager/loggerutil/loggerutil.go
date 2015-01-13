/* Datamanager-specific logging methods. */

package loggerutil

import (
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"log"
	"time"
)

// Assumes you haven't already called arvLogger.Edit()!
// If you have called arvLogger.Edit() this method will hang waiting
// for the lock you're already holding.
func FatalWithMessage(arvLogger *logger.Logger, message string) {
	if arvLogger != nil {
		properties, _ := arvLogger.Edit()
		properties["FATAL"] = message
		properties["run_info"].(map[string]interface{})["end_time"] = time.Now()
		arvLogger.ForceRecord()
	}

	log.Fatalf(message)
}
