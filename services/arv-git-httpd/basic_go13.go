// +build !go1.4

package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func BasicAuth(r *http.Request) (username, password string, ok bool) {
	toks := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(toks) != 2 || toks[0] != "Basic" {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(toks[1])
	if err != nil {
		return "", "", false
	}

	userAndPass := strings.SplitN(string(decoded), ":", 2)
	if len(userAndPass) != 2 {
		return "", "", false
	}

	return userAndPass[0], userAndPass[1], true
}
