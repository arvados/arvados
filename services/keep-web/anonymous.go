package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var anonymousTokens tokenSet

type tokenSet []string

func (ts *tokenSet) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if v && len(*ts) == 0 {
		*ts = append(*ts, os.Getenv("ARVADOS_API_TOKEN"))
	} else if !v {
		*ts = (*ts)[:0]
	}
	return err
}

func (ts *tokenSet) String() string {
	return fmt.Sprintf("%v", len(*ts) > 0)
}

func (ts *tokenSet) IsBoolFlag() bool {
	return true
}

func init() {
	flag.Var(&anonymousTokens, "allow-anonymous",
		"Serve public data to anonymous clients. Try the token supplied in the ARVADOS_API_TOKEN environment variable when none of the tokens provided in an HTTP request succeed in reading the desired collection.")
}
