package controller

import (
	"database/sql"
	"fmt"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

func (h *Handler) groupsShared(w http.ResponseWriter, req *http.Request, currentUser CurrentUser) {
	w.Write([]byte(fmt.Sprintf("Hello world %v\n", currentUser.UUID)))
}

func (h *Handler) handleGoAPI(w http.ResponseWriter, req *http.Request, next http.Handler) {
	if req.URL.Path != "/arvados/v1/groups/shared" {
		next.ServeHTTP(w, req)
		return
	}

	// Check token and get user UUID

	creds := auth.NewCredentials()
	creds.LoadTokensFromHTTPRequest(req)

	if len(creds.Tokens) == 0 {
		httpserver.Error(w, "Not logged in", http.StatusForbidden)
		return
	}

	currentUser := CurrentUser{Authorization: arvados.APIClientAuthorization{APIToken: creds.Tokens[0]}}
	err := h.validateAPItoken(req, &currentUser)
	if err != nil {
		if err == sql.ErrNoRows {
			httpserver.Error(w, "Not logged in", http.StatusForbidden)
		} else {
			httpserver.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Handle /arvados/v1/groups/shared

	h.groupsShared(w, req, currentUser)
}
