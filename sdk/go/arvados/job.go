// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Job is an arvados#job record
type Job struct {
	UUID                   string    `json:"uuid"`
	Etag                   string    `json:"etag"`
	OwnerUUID              string    `json:"owner_uuid"`
	ModifiedByClientUUID   string    `json:"modified_by_client_uuid"`
	ModifiedByUserUUID     string    `json:"modified_by_user_uuid"`
	ModifiedAt             time.Time `json:"modified_at"`
	SubmitID               string    `json:"submit_id"`
	Script                 string    `json:"script"`
	CancelledByClientUUID  string    `json:"cancelled_by_client_uuid"`
	CancelledByUserUUID    string    `json:"cancelled_by_user_uuid"`
	CancelledAt            time.Time `json:"cancelled_at"`
	StartedAt              time.Time `json:"started_at"`
	FinishedAt             time.Time `json:"finished_at"`
	Running                bool      `json:"running"`
	Success                bool      `json:"success"`
	Output                 string    `json:"output"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	IsLockedByUUID         string    `json:"is_locked_by_uuid"`
	Log                    string    `json:"log"`
	TasksSummary           string    `json:"tasks_summary"`
	RuntimeConstraints     string    `json:"runtime_constraints"`
	Nondeterministic       bool      `json:"nondeterministic"`
	Repository             string    `json:"repository"`
	SuppliedScriptVersion  string    `json:"supplied_script_version"`
	DockerImageLocator     string    `json:"docker_image_locator"`
	Priority               int       `json:"priority"`
	Description            string    `json:"description"`
	State                  string    `json:"state"`
	ArvadosSDKVersion      string    `json:"arvados_sdk_version"`
	Components             string    `json:"components"`
	ScriptParametersDigest string    `json:"script_parameters_digest"`
	WritableBy             []string  `json:"writable_by,omitempty"`
}

func (g Job) resourceName() string {
	return "job"
}
