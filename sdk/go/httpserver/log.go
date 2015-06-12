package httpserver

import (
	"log"
	"strings"
)

var escaper = strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\n", "\\n")

// Log calls log.Println but first transforms strings so they are
// safer to write in logs (e.g., 'foo"bar' becomes
// '"foo\"bar"'). Non-string args are left alone.
func Log(args ...interface{}) {
	newargs := make([]interface{}, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			newargs[i] = "\"" + escaper.Replace(s) + "\""
		} else {
			newargs[i] = arg
		}
	}
	log.Println(newargs...)
}
