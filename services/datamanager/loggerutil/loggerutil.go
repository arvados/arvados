/* Datamanager-specific logging methods. */

package loggerutil

import (
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"log"
	"time"
)

func FatalWithMessage(arvLogger *logger.Logger, message string) {
	if arvLogger != nil {
		arvLogger.FinalUpdate(func(p map[string]interface{}, e map[string]interface{}) {
			p["FATAL"] = message
			p["run_info"].(map[string]interface{})["time_finished"] = time.Now()
		})
	}

	log.Fatalf(message)
}
