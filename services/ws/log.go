package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func init() {
	log.SetFlags(0)
}

func errorLogf(f string, args ...interface{}) {
	log.Print(`{"error":`, string(mustMarshal(fmt.Sprintf(f, args...))), `}`)
}

var debugLogf = func(f string, args ...interface{}) {
	log.Print(`{"debug":`, string(mustMarshal(fmt.Sprintf(f, args...))), `}`)
}

func mustMarshal(v interface{}) []byte {
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return buf
}

func logj(args ...interface{}) {
	m := map[string]interface{}{"Time": time.Now().UTC()}
	for i := 0; i < len(args)-1; i += 2 {
		m[fmt.Sprintf("%s", args[i])] = args[i+1]
	}
	buf, err := json.Marshal(m)
	if err != nil {
		errorLogf("logj: %s", err)
		return
	}
	log.Print(string(buf))
}
