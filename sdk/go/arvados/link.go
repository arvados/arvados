// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// Link is an arvados#link record
type Link struct {
	UUID       string                 `json:"uuid,omiempty"`
	OwnerUUID  string                 `json:"owner_uuid"`
	Name       string                 `json:"name"`
	LinkClass  string                 `json:"link_class"`
	HeadUUID   string                 `json:"head_uuid"`
	HeadKind   string                 `json:"head_kind"`
	TailUUID   string                 `json:"tail_uuid"`
	TailKind   string                 `json:"tail_kind"`
	Properties map[string]interface{} `json:"properties"`
}

// LinkList is an arvados#linkList resource.
type LinkList struct {
	Items          []Link `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
}
