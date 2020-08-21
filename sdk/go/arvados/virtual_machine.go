// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// VirtualMachine is an arvados#virtualMachine resource.
type VirtualMachine struct {
	UUID               string     `json:"uuid"`
	OwnerUUID          string     `json:"owner_uuid"`
	Hostname           string     `json:"hostname"`
	CreatedAt          *time.Time `json:"created_at"`
	ModifiedAt         *time.Time `json:"modified_at"`
	ModifiedByUserUUID string     `json:"modified_by_user_uuid"`
}

// VirtualMachineList is an arvados#virtualMachineList resource.
type VirtualMachineList struct {
	Items          []VirtualMachine `json:"items"`
	ItemsAvailable int              `json:"items_available"`
	Offset         int              `json:"offset"`
	Limit          int              `json:"limit"`
}
