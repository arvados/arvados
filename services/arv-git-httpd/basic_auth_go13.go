// +build !go1.4

package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func BasicAuth(r *http.Request) (username, password string, ok bool) {
	tokens := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(tokens) != 2 || tokens[0] != "Basic" {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(tokens[1])
	if err != nil {
		return "", "", false
	}

	userAndPass := strings.SplitN(string(decoded), ":", 2)
	if len(userAndPass) != 2 {
		return "", "", false
	}

	return userAndPass[0], userAndPass[1], true
}
