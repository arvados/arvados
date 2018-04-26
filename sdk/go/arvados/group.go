// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// Group is an arvados#group record
type Group struct {
	UUID       string `json:"uuid,omitempty"`
	Name       string `json:"name,omitempty"`
	OwnerUUID  string `json:"owner_uuid,omitempty"`
	GroupClass string `json:"group_class"`
}

// GroupList is an arvados#groupList resource.
type GroupList struct {
	Items          []Group `json:"items"`
	ItemsAvailable int     `json:"items_available"`
	Offset         int     `json:"offset"`
	Limit          int     `json:"limit"`
}

func (g Group) resourceName() string {
	return "group"
}
