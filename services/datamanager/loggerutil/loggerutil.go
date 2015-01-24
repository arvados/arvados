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
		arvLogger.ForceUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			p["FATAL"] = message
			p["run_info"].(map[string]interface{})["end_time"] = time.Now()
		})
	}

	log.Fatalf(message)
}
