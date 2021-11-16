// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Link is an arvados#link record
type Link struct {
	UUID                 string                 `json:"uuid,omitempty"`
	Etag                 string                 `json:"etag"`
	Href                 string                 `json:"href"`
	OwnerUUID            string                 `json:"owner_uuid"`
	Name                 string                 `json:"name"`
	LinkClass            string                 `json:"link_class"`
	CreatedAt            time.Time              `json:"created_at"`
	ModifiedAt           time.Time              `json:"modified_at"`
	ModifiedByClientUUID string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID   string                 `json:"modified_by_user_uuid"`
	HeadUUID             string                 `json:"head_uuid"`
	HeadKind             string                 `json:"head_kind"`
	TailUUID             string                 `json:"tail_uuid"`
	TailKind             string                 `json:"tail_kind"`
	Properties           map[string]interface{} `json:"properties"`
}

// LinkList is an arvados#linkList resource.
type LinkList struct {
	Items          []Link `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
}
