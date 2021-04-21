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

	"github.com/gorilla/mux"
)

func guessAndParse(k, v string) (interface{}, error) {
	// All of these form values arrive as strings, so we need some
	// type-guessing to accept non-string inputs:
	//
	// Values for parameters that take ints (limit=1) or bools
	// (include_trash=1) are parsed accordingly.
	//
	// "null" and "" are nil.
	//
	// Values that look like JSON objects, arrays, or strings are
	// parsed as JSON.
	//
	// The rest are left as strings.
	switch {
	case intParams[k]:
		return strconv.ParseInt(v, 10, 64)
	case boolParams[k]:
		return stringToBool(v), nil
	case v == "null" || v == "":
		return nil, nil
	case strings.HasPrefix(v, "["):
		var j []interface{}
		err := json.Unmarshal([]byte(v), &j)
		return j, err
	case strings.HasPrefix(v, "{"):
		var j map[string]interface{}
		err := json.Unmarshal([]byte(v), &j)
		return j, err
	case strings.HasPrefix(v, "\""):
		var j string
		err := json.Unmarshal([]byte(v), &j)
		return j, err
	default:
		return v, nil
	}
	// TODO: Need to accept "?foo[]=bar&foo[]=baz" as
	// foo=["bar","baz"]?
}

// Parse req as an Arvados V1 API request and return the request
// parameters.
//
// If the request has a parameter whose name is attrsKey (e.g.,
// "collection"), it is renamed to "attrs".
func (rtr *router) loadRequestParams(req *http.Request, attrsKey string) (map[string]interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		if err.Error() == "http: request body too large" {
			return nil, httpError(http.StatusRequestEntityTooLarge, err)
		} else {
			return nil, httpError(http.StatusBadRequest, err)
		}
	}
	params := map[string]interface{}{}

	// Load parameters from req.Form, which (after
	// req.ParseForm()) includes the query string and -- when
	// Content-Type is application/x-www-form-urlencoded -- the
	// request body.
	for k, values := range req.Form {
		for _, v := range values {
			params[k], err = guessAndParse(k, v)
			if err != nil {
				return nil, err
			}
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
			switch v := v.(type) {
			case string:
				// The Ruby "arv" cli tool sends a
				// JSON-encode params map with
				// JSON-encoded values.
				dec, err := guessAndParse(k, v)
				if err != nil {
					return nil, err
				}
				jsonParams[k] = dec
				params[k] = dec
			default:
				params[k] = v
			}
		}
		if attrsKey != "" && params[attrsKey] == nil {
			// Copy top-level parameters from JSON request
			// body into params[attrsKey]. Some SDKs rely
			// on this Rails API feature; see
			// https://api.rubyonrails.org/v5.2.1/classes/ActionController/ParamsWrapper.html
			params[attrsKey] = jsonParams
		}
	}

	for k, v := range mux.Vars(req) {
		params[k] = v
	}

	if v, ok := params[attrsKey]; ok && attrsKey != "" {
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
	"distinct":                true,
	"ensure_unique_name":      true,
	"include_trash":           true,
	"include_old_versions":    true,
	"redirect_to_new_user":    true,
	"send_notification_email": true,
	"bypass_federation":       true,
	"recursive":               true,
	"exclude_home_project":    true,
}

func stringToBool(s string) bool {
	switch s {
	case "", "false", "0":
		return false
	default:
		return true
	}
}
