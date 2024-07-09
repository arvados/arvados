// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// AuthorizedKey is an arvados#authorizedKey resource.
type AuthorizedKey struct {
	UUID               string    `json:"uuid"`
	Etag               string    `json:"etag"`
	OwnerUUID          string    `json:"owner_uuid"`
	CreatedAt          time.Time `json:"created_at"`
	ModifiedAt         time.Time `json:"modified_at"`
	ModifiedByUserUUID string    `json:"modified_by_user_uuid"`
	Name               string    `json:"name"`
	AuthorizedUserUUID string    `json:"authorized_user_uuid"`
	PublicKey          string    `json:"public_key"`
	KeyType            string    `json:"key_type"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// AuthorizedKeyList is an arvados#authorizedKeyList resource.
type AuthorizedKeyList struct {
	Items          []AuthorizedKey `json:"items"`
	ItemsAvailable int             `json:"items_available"`
	Offset         int             `json:"offset"`
	Limit          int             `json:"limit"`
}
