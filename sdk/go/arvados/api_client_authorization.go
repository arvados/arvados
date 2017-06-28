// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

// APIClientAuthorization is an arvados#apiClientAuthorization resource.
type APIClientAuthorization struct {
	UUID     string `json:"uuid"`
	APIToken string `json:"api_token"`
}

// APIClientAuthorizationList is an arvados#apiClientAuthorizationList resource.
type APIClientAuthorizationList struct {
	Items []APIClientAuthorization `json:"items"`
}
