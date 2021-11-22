// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// APIClientAuthorization is an arvados#apiClientAuthorization resource.
type APIClientAuthorization struct {
	UUID                 string    `json:"uuid"`
	APIClientID          int       `json:"api_client_id"`
	APIToken             string    `json:"api_token"`
	CreatedAt            time.Time `json:"created_at"`
	CreatedByIPAddress   *string   `json:"created_by_ip_address"`
	DefaultOwnerUUID     *string   `json:"default_owner_uuid"`
	Etag                 string    `json:"etag"`
	ExpiresAt            string    `json:"expires_at"`
	Href                 string    `json:"href"`
	LastUsedAt           time.Time `json:"last_used_at"`
	LastUsedByIPAddress  *string   `json:"last_used_by_ip_address"`
	ModifiedAt           time.Time `json:"modified_at"`
	ModifiedByClientUUID *string   `json:"modified_by_client_uuid"`
	ModifiedByUserUUID   *string   `json:"modified_by_user_uuid"`
	OwnerUUID            string    `json:"owner_uuid"`
	Scopes               []string  `json:"scopes"`
	UserID               int       `json:"user_id"`
}

// APIClientAuthorizationList is an arvados#apiClientAuthorizationList resource.
type APIClientAuthorizationList struct {
	Items []APIClientAuthorization `json:"items"`
}

func (aca APIClientAuthorization) TokenV2() string {
	return "v2/" + aca.UUID + "/" + aca.APIToken
}
