// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// Link is an arvados#link record
type Link struct {
	UUID      string `json:"uuid,omiempty"`
	OwnerUUID string `json:"owner_uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	LinkClass string `json:"link_class,omitempty"`
	HeadUUID  string `json:"head_uuid,omitempty"`
	HeadKind  string `json:"head_kind,omitempty"`
	TailUUID  string `json:"tail_uuid,omitempty"`
	TailKind  string `json:"tail_kind,omitempty"`
}

// UserList is an arvados#userList resource.
type LinkList struct {
	Items          []Link `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
}
