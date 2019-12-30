// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// User is an arvados#user record
type User struct {
	UUID                 string                 `json:"uuid"`
	Etag                 string                 `json:"etag"`
	IsActive             bool                   `json:"is_active"`
	IsAdmin              bool                   `json:"is_admin"`
	Username             string                 `json:"username"`
	Email                string                 `json:"email"`
	FullName             string                 `json:"full_name"`
	FirstName            string                 `json:"first_name"`
	LastName             string                 `json:"last_name"`
	IdentityURL          string                 `json:"identity_url"`
	IsInvited            bool                   `json:"is_invited"`
	OwnerUUID            string                 `json:"owner_uuid"`
	CreatedAt            time.Time              `json:"created_at"`
	ModifiedAt           time.Time              `json:"modified_at"`
	ModifiedByUserUUID   string                 `json:"modified_by_user_uuid"`
	ModifiedByClientUUID string                 `json:"modified_by_client_uuid"`
	Prefs                map[string]interface{} `json:"prefs"`
	WritableBy           []string               `json:"writable_by,omitempty"`
}

// UserList is an arvados#userList resource.
type UserList struct {
	Items          []User `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
}

// CurrentUser calls arvados.v1.users.current, and returns the User
// record corresponding to this client's credentials.
func (c *Client) CurrentUser() (User, error) {
	var u User
	err := c.RequestAndDecode(&u, "GET", "arvados/v1/users/current", nil, nil)
	return u, err
}
