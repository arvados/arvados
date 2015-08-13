package httpserver

import (
	"fmt"
	"log"
)

// Log calls log.Println but first transforms strings so they are
// safer to write in logs (e.g., 'foo"bar' becomes
// '"foo\"bar"'). Arguments that aren't strings and don't have a
// (String() string) method are left alone.
func Log(args ...interface{}) {
	newargs := make([]interface{}, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			newargs[i] = fmt.Sprintf("%+q", s)
		} else if s, ok := arg.(fmt.Stringer); ok {
			newargs[i] = fmt.Sprintf("%+q", s.String())
		} else {
			newargs[i] = arg
		}
	}
	log.Println(newargs...)
}
