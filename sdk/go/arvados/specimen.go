// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

type Specimen struct {
	UUID       string                 `json:"uuid"`
	OwnerUUID  string                 `json:"owner_uuid"`
	CreatedAt  time.Time              `json:"created_at"`
	ModifiedAt time.Time              `json:"modified_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Properties map[string]interface{} `json:"properties"`
}

type SpecimenList struct {
	Items          []Specimen `json:"items"`
	ItemsAvailable int        `json:"items_available"`
	Offset         int        `json:"offset"`
	Limit          int        `json:"limit"`
}
