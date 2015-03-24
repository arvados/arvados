// +build go1.4

package main

import (
	"net/http"
)

func BasicAuth(r *http.Request) (username, password string, ok bool) {
	return r.BasicAuth()
}
