// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Credential is an arvados#credential record
type Credential struct {
	UUID               string    `json:"uuid,omitempty"`
	Etag               string    `json:"etag"`
	OwnerUUID          string    `json:"owner_uuid"`
	CreatedAt          time.Time `json:"created_at"`
	ModifiedAt         time.Time `json:"modified_at"`
	ModifiedByUserUUID string    `json:"modified_by_user_uuid"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	CredentialClass    string    `json:"credential_class"`
	CredentialId       string    `json:"credential_id"`
	CredentialSecret   string    `json:"credential_secret,omitempty"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// CredentialList is an arvados#credentialList resource.
type CredentialList struct {
	Items          []Credential `json:"items"`
	ItemsAvailable int          `json:"items_available"`
	Offset         int          `json:"offset"`
	Limit          int          `json:"limit"`
}
