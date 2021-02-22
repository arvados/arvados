// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

type responseOptions struct {
	Select []string
	Count  string
}

func (rtr *router) responseOptions(opts interface{}) (responseOptions, error) {
	var rOpts responseOptions
	switch opts := opts.(type) {
	case *arvados.GetOptions:
		rOpts.Select = opts.Select
	case *arvados.ListOptions:
		rOpts.Select = opts.Select
		rOpts.Count = opts.Count
	}
	return rOpts, nil
}

func applySelectParam(selectParam []string, orig map[string]interface{}) map[string]interface{} {
	if len(selectParam) == 0 {
		return orig
	}
	selected := map[string]interface{}{}
	for _, attr := range selectParam {
		if v, ok := orig[attr]; ok {
			selected[attr] = v
		}
	}
	// Some keys are always preserved, even if not requested
	for _, k := range []string{"etag", "kind", "writable_by"} {
		if v, ok := orig[k]; ok {
			selected[k] = v
		}
	}
	return selected
}

func (rtr *router) sendResponse(w http.ResponseWriter, req *http.Request, resp interface{}, opts responseOptions) {
	var tmp map[string]interface{}

	if resp, ok := resp.(http.Handler); ok {
		// resp knows how to write its own http response
		// header and body.
		resp.ServeHTTP(w, req)
		return
	}

	err := rtr.transcode(resp, &tmp)
	if err != nil {
		rtr.sendError(w, err)
		return
	}

	respKind := kind(resp)
	if respKind != "" {
		tmp["kind"] = respKind
	}
	if included, ok := tmp["included"]; ok && included == nil {
		tmp["included"] = make([]interface{}, 0)
	}

	defaultItemKind := ""
	if strings.HasSuffix(respKind, "List") {
		defaultItemKind = strings.TrimSuffix(respKind, "List")
	}

	if items, ok := tmp["items"].([]interface{}); ok {
		for i, item := range items {
			// Fill in "kind" by inspecting UUID/PDH if
			// possible; fall back on assuming each
			// Items[] entry in an "arvados#fooList"
			// response should have kind="arvados#foo".
			item, _ := item.(map[string]interface{})
			infix := ""
			if uuid, _ := item["uuid"].(string); len(uuid) == 27 {
				infix = uuid[6:11]
			}
			if k := kind(infixMap[infix]); k != "" {
				item["kind"] = k
			} else if pdh, _ := item["portable_data_hash"].(string); pdh != "" {
				item["kind"] = "arvados#collection"
			} else if defaultItemKind != "" {
				item["kind"] = defaultItemKind
			}
			item = applySelectParam(opts.Select, item)
			rtr.mungeItemFields(item)
			items[i] = item
		}
		if opts.Count == "none" {
			delete(tmp, "items_available")
		}
	} else {
		tmp = applySelectParam(opts.Select, tmp)
		rtr.mungeItemFields(tmp)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(tmp)
}

func (rtr *router) sendError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err, ok := err.(interface{ HTTPStatus() int }); ok {
		code = err.HTTPStatus()
	}
	httpserver.Error(w, err.Error(), code)
}

var infixMap = map[string]interface{}{
	"4zz18": arvados.Collection{},
	"xvhdp": arvados.ContainerRequest{},
	"dz642": arvados.Container{},
	"j7d0g": arvados.Group{},
	"8i9sb": arvados.Job{},
	"d1hrv": arvados.PipelineInstance{},
	"p5p6p": arvados.PipelineTemplate{},
	"j58dm": arvados.Specimen{},
	"q1cn2": arvados.Trait{},
	"7fd4e": arvados.Workflow{},
}

var mungeKind = regexp.MustCompile(`\..`)

func kind(resp interface{}) string {
	t := fmt.Sprintf("%T", resp)
	if !strings.HasPrefix(t, "arvados.") {
		return ""
	}
	return mungeKind.ReplaceAllStringFunc(t, func(s string) string {
		// "arvados.CollectionList" => "arvados#collectionList"
		return "#" + strings.ToLower(s[1:])
	})
}

func (rtr *router) mungeItemFields(tmp map[string]interface{}) {
	for k, v := range tmp {
		if strings.HasSuffix(k, "_at") {
			// Format non-nil timestamps as
			// rfc3339NanoFixed (otherwise they would use
			// the default time encoding, which omits
			// trailing zeroes).
			switch tv := v.(type) {
			case *time.Time:
				if tv == nil || tv.IsZero() {
					tmp[k] = nil
				} else {
					tmp[k] = tv.Format(rfc3339NanoFixed)
				}
			case time.Time:
				if tv.IsZero() {
					tmp[k] = nil
				} else {
					tmp[k] = tv.Format(rfc3339NanoFixed)
				}
			case string:
				if tv == "" {
					tmp[k] = nil
				} else if t, err := time.Parse(time.RFC3339Nano, tv); err != nil {
					// pass through an invalid time value (?)
				} else if t.IsZero() {
					tmp[k] = nil
				} else {
					tmp[k] = t.Format(rfc3339NanoFixed)
				}
			}
		}
		// Arvados API spec says when these fields are empty
		// they appear in responses as null, rather than a
		// zero value.
		switch k {
		case "output_uuid", "output_name", "log_uuid", "description", "requesting_container_uuid", "container_uuid":
			if v == "" {
				tmp[k] = nil
			}
		case "container_count_max":
			if v == float64(0) {
				tmp[k] = nil
			}
		}
	}
}
