// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Trait is an arvados#trait record
type Trait struct {
	UUID                 string                 `json:"uuid"`
	Etag                 string                 `json:"etag"`
	OwnerUUID            string                 `json:"owner_uuid"`
	CreatedAt            time.Time              `json:"created_at"`
	ModifiedByClientUUID string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID   string                 `json:"modified_by_user_uuid"`
	ModifiedAt           time.Time              `json:"modified_at"`
	Name                 string                 `json:"name"`
	Properties           map[string]interface{} `json:"properties"`
	UpdatedAt            time.Time              `json:"updated_at"`
	WritableBy           []string               `json:"writable_by,omitempty"`
}

func (g Trait) resourceName() string {
	return "trait"
}
