package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

	// select groups that are readable by current user AND
	//   the owner_uuid is a user (but not the current user) OR
	//   the owner_uuid is not readable by the current user
	//   the owner_uuid group_class is not a project

	baseQuery := `SELECT %s from groups
WHERE
  EXISTS(SELECT 1 from materialized_permission_view WHERE user_uuid=$1 AND target_uuid=groups.uuid) AND
  (groups.owner_uuid IN (SELECT uuid FROM users WHERE users.uuid != $1) OR
    NOT EXISTS(SELECT 1 FROM materialized_permission_view WHERE user_uuid=$1 AND target_uuid=groups.owner_uuid) OR
    EXISTS(SELECT 1 FROM groups as gp where gp.uuid=groups.owner_uuid and gp.group_class != 'project'))
LIMIT 50`

	err = db.QueryRowContext(req.Context(), fmt.Sprintf(baseQuery, "count(uuid)"), currentUser.UUID).Scan(&gl.ItemsAvailable)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(req.Context(), fmt.Sprintf(baseQuery, "uuid, name, owner_uuid, group_class"), currentUser.UUID)
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
