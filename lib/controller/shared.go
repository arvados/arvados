package controller

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

func (h *Handler) groupsShared(w http.ResponseWriter, req *http.Request, currentUser CurrentUser) error {

	db, err := h.db(req)
	if err != nil {
		return err
	}

	gl := arvados.GroupList{}

	err = db.QueryRowContext(req.Context(), `SELECT count(uuid) from groups`).Scan(&gl.ItemsAvailable)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(req.Context(), `SELECT uuid, name, owner_uuid, group_class from groups limit 50`)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var g arvados.Group
		rows.Scan(&g.UUID, &g.Name, &g.OwnerUUID, &g.GroupClass)
		gl.Items = append(gl.Items, g)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(gl)
	if err != nil {
		return err
	}

	return nil
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

	err = h.groupsShared(w, req, currentUser)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadRequest)
	}
}
