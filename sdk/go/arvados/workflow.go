// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Workflow is an arvados#workflow resource.
type Workflow struct {
	UUID        string     `json:"uuid"`
	OwnerUUID   string     `json:"owner_uuid"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Definition  string     `json:"definition"`
	CreatedAt   *time.Time `json:"created_at"`
	ModifiedAt  *time.Time `json:"modified_at"`
}

// WorkflowList is an arvados#workflowList resource.
type WorkflowList struct {
	Items          []Workflow `json:"items"`
	ItemsAvailable int        `json:"items_available"`
	Offset         int        `json:"offset"`
	Limit          int        `json:"limit"`
}
