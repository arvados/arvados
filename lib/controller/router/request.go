// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"fmt"
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

	// Load parameters from req.Form, which (after
	// req.ParseForm()) includes the query string and -- when
	// Content-Type is application/x-www-form-urlencoded -- the
	// request body.
	for k, values := range req.Form {
		// All of these form values arrive as strings, so we
		// need some type-guessing to accept non-string
		// inputs:
		//
		// Values for parameters that take ints (limit=1) or
		// bools (include_trash=1) are parsed accordingly.
		//
		// "null" and "" are nil.
		//
		// Values that look like JSON objects, arrays, or
		// strings are parsed as JSON.
		//
		// The rest are left as strings.
		for _, v := range values {
			switch {
			case intParams[k]:
				params[k], err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, err
				}
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
			default:
				params[k] = v
			}
			// TODO: Need to accept "?foo[]=bar&foo[]=baz"
			// as foo=["bar","baz"]?
		}
	}

	// Decode body as JSON if Content-Type request header is
	// missing or application/json.
	mt := req.Header.Get("Content-Type")
	if ct, _, err := mime.ParseMediaType(mt); err != nil && mt != "" {
		return nil, fmt.Errorf("error parsing media type %q: %s", mt, err)
	} else if (ct == "application/json" || mt == "") && req.ContentLength != 0 {
		jsonParams := map[string]interface{}{}
		err := json.NewDecoder(req.Body).Decode(&jsonParams)
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
		if v, ok := v.(map[string]interface{}); ok {
			// Delete field(s) that appear in responses
			// but not in update attrs, so clients can
			// fetch-modify-update.
			delete(v, "etag")
			delete(v, "unsigned_manifest_text")
		}
		params["attrs"] = v
		delete(params, attrsKey)
	}

	if order, ok := params["order"].(string); ok {
		// We must accept strings ("foo, bar desc") and arrays
		// (["foo", "bar desc"]) because RailsAPI does.
		// Convert to an array here before trying to unmarshal
		// into options structs.
		if order == "" {
			delete(params, "order")
		} else {
			params["order"] = strings.Split(order, ",")
		}
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

var intParams = map[string]bool{
	"limit":  true,
	"offset": true,
}

var boolParams = map[string]bool{
	"distinct":             true,
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
