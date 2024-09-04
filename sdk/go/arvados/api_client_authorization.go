// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// APIClientAuthorization is an arvados#apiClientAuthorization resource.
type APIClientAuthorization struct {
	UUID                string    `json:"uuid"`
	APIToken            string    `json:"api_token"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedByIPAddress  string    `json:"created_by_ip_address"`
	Etag                string    `json:"etag"`
	ExpiresAt           time.Time `json:"expires_at"`
	LastUsedAt          time.Time `json:"last_used_at"`
	LastUsedByIPAddress string    `json:"last_used_by_ip_address"`
	ModifiedAt          time.Time `json:"modified_at"`
	ModifiedByUserUUID  string    `json:"modified_by_user_uuid"`
	OwnerUUID           string    `json:"owner_uuid"`
	Scopes              []string  `json:"scopes"`
}

// APIClientAuthorizationList is an arvados#apiClientAuthorizationList resource.
type APIClientAuthorizationList struct {
	Items []APIClientAuthorization `json:"items"`
}

func (aca APIClientAuthorization) TokenV2() string {
	return "v2/" + aca.UUID + "/" + aca.APIToken
}
