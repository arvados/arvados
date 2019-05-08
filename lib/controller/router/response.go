// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

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
	// Format non-nil timestamps as rfc3339NanoFixed (by default
	// they will have been encoded to time.RFC3339Nano, which
	// omits trailing zeroes).
	for k, v := range tmp {
		if !strings.HasSuffix(k, "_at") {
			continue
		}
		switch tv := v.(type) {
		case *time.Time:
			if tv == nil {
				break
			}
			tmp[k] = tv.Format(rfc3339NanoFixed)
		case time.Time:
			tmp[k] = tv.Format(rfc3339NanoFixed)
		case string:
			t, err := time.Parse(time.RFC3339Nano, tv)
			if err != nil {
				break
			}
			tmp[k] = t.Format(rfc3339NanoFixed)
		}
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
