// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"time"
)

// Log is an arvados#log record
type Log struct {
	ID              uint64                 `json:"id,omitempty"`
	UUID            string                 `json:"uuid,omitempty"`
	ObjectUUID      string                 `json:"object_uuid,omitempty"`
	ObjectOwnerUUID string                 `json:"object_owner_uuid,omitempty"`
	EventType       string                 `json:"event_type,omitempty"`
	EventAt         *time.Time             `json:"event,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	CreatedAt       *time.Time             `json:"created_at,omitempty"`
}

// LogList is an arvados#logList resource.
type LogList struct {
	Items          []Log `json:"items"`
	ItemsAvailable int   `json:"items_available"`
	Offset         int   `json:"offset"`
	Limit          int   `json:"limit"`
}
