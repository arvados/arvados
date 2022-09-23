// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"fmt"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// GroupCreate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) GroupCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.Group, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Group{}, err
	}
	resp, err := conn.railsProxy.GroupCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (conn *Conn) GroupGet(ctx context.Context, opts arvados.GetOptions) (arvados.Group, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.GroupGet(ctx, opts)
}

// GroupUpdate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) GroupUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.Group, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Group{}, err
	}
	resp, err := conn.railsProxy.GroupUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (conn *Conn) GroupList(ctx context.Context, opts arvados.ListOptions) (arvados.GroupList, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.GroupList(ctx, opts)
}

func (conn *Conn) GroupDelete(ctx context.Context, opts arvados.DeleteOptions) (arvados.Group, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.GroupDelete(ctx, opts)
}

func (conn *Conn) GroupContents(ctx context.Context, options arvados.GroupContentsOptions) (arvados.ObjectList, error) {
	conn.logActivity(ctx)

	// The requested UUID can be a user (virtual home project), which we just pass on to
	// the API server.
	if strings.Index(options.UUID, "-j7d0g-") != 5 {
		return conn.railsProxy.GroupContents(ctx, options)
	}

	var resp arvados.ObjectList

	// Get the group object
	respGroup, err := conn.GroupGet(ctx, arvados.GetOptions{UUID: options.UUID})
	if err != nil {
		return resp, err
	}

	// If the group has groupClass 'filter', apply the filters before getting the contents.
	if respGroup.GroupClass == "filter" {
		if filters, ok := respGroup.Properties["filters"].([]interface{}); ok {
			for _, f := range filters {
				// f is supposed to be a []string
				tmp, ok2 := f.([]interface{})
				if !ok2 || len(tmp) < 3 {
					return resp, fmt.Errorf("filter unparsable: %T, %+v, original field: %T, %+v\n", tmp, tmp, f, f)
				}
				var filter arvados.Filter
				if attr, ok2 := tmp[0].(string); ok2 {
					filter.Attr = attr
				} else {
					return resp, fmt.Errorf("filter unparsable: attribute must be string: %T, %+v, filter: %T, %+v\n", tmp[0], tmp[0], f, f)
				}
				if operator, ok2 := tmp[1].(string); ok2 {
					filter.Operator = operator
				} else {
					return resp, fmt.Errorf("filter unparsable: operator must be string: %T, %+v, filter: %T, %+v\n", tmp[1], tmp[1], f, f)
				}
				filter.Operand = tmp[2]
				options.Filters = append(options.Filters, filter)
			}
		} else {
			return resp, fmt.Errorf("filter unparsable: not an array\n")
		}
		// Use the generic /groups/contents endpoint for filter groups
		options.UUID = ""
	}

	return conn.railsProxy.GroupContents(ctx, options)
}
