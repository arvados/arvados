// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Workflow is an arvados#workflow resource.
type Workflow struct {
	UUID        string     `json:"uuid,omitempty"`
	OwnerUUID   string     `json:"owner_uuid,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Definition  string     `json:"definition,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	ModifiedAt  *time.Time `json:"modified_at,omitempty"`
}

// WorkflowList is an arvados#workflowList resource.
type WorkflowList struct {
	Items          []Workflow `json:"items"`
	ItemsAvailable int        `json:"items_available"`
	Offset         int        `json:"offset"`
	Limit          int        `json:"limit"`
}
