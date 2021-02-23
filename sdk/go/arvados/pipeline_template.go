// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// PipelineTemplate is an arvados#pipelineTemplate record
type PipelineTemplate struct {
	UUID                 string    `json:"uuid"`
	Etag                 string    `json:"etag"`
	OwnerUUID            string    `json:"owner_uuid"`
	CreatedAt            time.Time `json:"created_at"`
	ModifiedByClientUUID string    `json:"modified_by_client_uuid"`
	ModifiedByUserUUID   string    `json:"modified_by_user_uuid"`
	ModifiedAt           time.Time `json:"modified_at"`
	Name                 string    `json:"name"`
	Components           string    `json:"components"`
	UpdatedAt            time.Time `json:"updated_at"`
	Description          string    `json:"description"`
	WritableBy           []string  `json:"writable_by,omitempty"`
}

func (g PipelineTemplate) resourceName() string {
	return "pipelineTemplate"
}
