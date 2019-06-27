// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// Group is an arvados#group record
type Group struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	OwnerUUID  string `json:"owner_uuid"`
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
