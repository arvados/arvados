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
		"Try using the specified token when a client does not provide a valid token. If this flag is used multiple times, each token will be tried in turn until one works.")
}
