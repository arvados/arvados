// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type responseOptions struct {
	Select []string
}

func (rtr *router) responseOptions(opts interface{}) (responseOptions, error) {
	var rOpts responseOptions
	switch opts := opts.(type) {
	case *arvados.GetOptions:
		rOpts.Select = opts.Select
	}
	return rOpts, nil
}

func (rtr *router) sendResponse(w http.ResponseWriter, resp interface{}, opts responseOptions) {
	var tmp map[string]interface{}
	err := rtr.transcode(resp, &tmp)
	if err != nil {
		rtr.sendError(w, err)
		return
	}
	if len(opts.Select) > 0 {
		selected := map[string]interface{}{}
		for _, attr := range opts.Select {
			if v, ok := tmp[attr]; ok {
				selected[attr] = v
			}
		}
		tmp = selected
	}
	json.NewEncoder(w).Encode(tmp)
}

func (rtr *router) sendError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err, ok := err.(interface{ HTTPStatus() int }); ok {
		code = err.HTTPStatus()
	}
	httpserver.Error(w, err.Error(), code)
}
