// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"time"
)

// Log is an arvados#log record
type Log struct {
	ID              uint64                 `json:"id"`
	UUID            string                 `json:"uuid"`
	OwnerUUID       string                 `json:"owner_uuid"`
	ObjectUUID      string                 `json:"object_uuid"`
	ObjectOwnerUUID string                 `json:"object_owner_uuid"`
	EventType       string                 `json:"event_type"`
	EventAt         time.Time              `json:"event"`
	Summary         string                 `json:"summary"`
	Properties      map[string]interface{} `json:"properties"`
	CreatedAt       time.Time              `json:"created_at"`
	ModifiedAt      time.Time              `json:"modified_at"`
}

// LogList is an arvados#logList resource.
type LogList struct {
	Items          []Log `json:"items"`
	ItemsAvailable int   `json:"items_available"`
	Offset         int   `json:"offset"`
	Limit          int   `json:"limit"`
}
