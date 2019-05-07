// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Parse req as an Arvados V1 API request and return the request
// parameters.
//
// If the request has a parameter whose name is attrsKey (e.g.,
// "collection"), it is renamed to "attrs".
func (rtr *router) loadRequestParams(req *http.Request, attrsKey string) (map[string]interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, httpError(http.StatusBadRequest, err)
	}
	params := map[string]interface{}{}
	for k, values := range req.Form {
		for _, v := range values {
			switch {
			case boolParams[k]:
				params[k] = stringToBool(v)
			case v == "null" || v == "":
				params[k] = nil
			case strings.HasPrefix(v, "["):
				var j []interface{}
				err := json.Unmarshal([]byte(v), &j)
				if err != nil {
					return nil, err
				}
				params[k] = j
			case strings.HasPrefix(v, "{"):
				var j map[string]interface{}
				err := json.Unmarshal([]byte(v), &j)
				if err != nil {
					return nil, err
				}
				params[k] = j
			case strings.HasPrefix(v, "\""):
				var j string
				err := json.Unmarshal([]byte(v), &j)
				if err != nil {
					return nil, err
				}
				params[k] = j
			case k == "limit" || k == "offset":
				params[k], err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, err
				}
			default:
				params[k] = v
			}
			// TODO: Need to accept "?foo[]=bar&foo[]=baz"
			// as foo=["bar","baz"]?
		}
	}
	if ct, _, err := mime.ParseMediaType(req.Header.Get("Content-Type")); err != nil && ct == "application/json" {
		jsonParams := map[string]interface{}{}
		err := json.NewDecoder(req.Body).Decode(jsonParams)
		if err != nil {
			return nil, httpError(http.StatusBadRequest, err)
		}
		for k, v := range jsonParams {
			params[k] = v
		}
		if attrsKey != "" && params[attrsKey] == nil {
			// Copy top-level parameters from JSON request
			// body into params[attrsKey]. Some SDKs rely
			// on this Rails API feature; see
			// https://api.rubyonrails.org/v5.2.1/classes/ActionController/ParamsWrapper.html
			params[attrsKey] = jsonParams
		}
	}

	routeParams, _ := req.Context().Value(httprouter.ParamsKey).(httprouter.Params)
	for _, p := range routeParams {
		params[p.Key] = p.Value
	}

	if v, ok := params[attrsKey]; ok && attrsKey != "" {
		params["attrs"] = v
		delete(params, attrsKey)
	}
	return params, nil
}

// Copy src to dst, using json as an intermediate format in order to
// invoke src's json-marshaling and dst's json-unmarshaling behaviors.
func (rtr *router) transcode(src interface{}, dst interface{}) error {
	var errw error
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		errw = json.NewEncoder(pw).Encode(src)
	}()
	defer pr.Close()
	err := json.NewDecoder(pr).Decode(dst)
	if errw != nil {
		return errw
	}
	return err
}

var boolParams = map[string]bool{
	"ensure_unique_name":   true,
	"include_trash":        true,
	"include_old_versions": true,
}

func stringToBool(s string) bool {
	switch s {
	case "", "false", "0":
		return false
	default:
		return true
	}
}
