package main

import (
	"flag"
	"fmt"
)

var anonymousTokens tokenSet

type tokenSet []string

func (ts *tokenSet) Set(t string) error {
	*ts = append(*ts, t)
	return nil
}

func (ts *tokenSet) String() string {
	return fmt.Sprintf("%+v", (*ts)[:])
}

func init() {
	flag.Var(&anonymousTokens, "anonymous-token",
		"API token to try when none of the tokens provided in an HTTP request succeed in reading the desired collection. If this flag is used more than once, each token will be attempted in turn until one works.")
}
