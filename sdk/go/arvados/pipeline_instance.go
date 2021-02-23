// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// PipelineInstance is an arvados#pipelineInstance record
type PipelineInstance struct {
	UUID                 string                 `json:"uuid"`
	Etag                 string                 `json:"etag"`
	OwnerUUID            string                 `json:"owner_uuid"`
	CreatedAt            time.Time              `json:"created_at"`
	ModifiedByClientUUID string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID   string                 `json:"modified_by_user_uuid"`
	ModifiedAt           time.Time              `json:"modified_at"`
	PipelineTemplateUUID string                 `json:"pipeline_template_uuid"`
	Name                 string                 `json:"name"`
	Components           map[string]interface{} `json:"components"`
	UpdatedAt            time.Time              `json:"updated_at"`
	Properties           map[string]interface{} `json:"properties"`
	State                string                 `json:"state"`
	ComponentsSummary    map[string]interface{} `json:"components_summary"`
	StartedAt            time.Time              `json:"started_at"`
	FinishedAt           time.Time              `json:"finished_at"`
	Description          string                 `json:"description"`
	WritableBy           []string               `json:"writable_by,omitempty"`
}

func (g PipelineInstance) resourceName() string {
	return "pipelineInstance"
}
