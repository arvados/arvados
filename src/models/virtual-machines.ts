// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "~/models/resource";

export interface VirtualMachinesResource extends Resource {
    hostname: string;
}

export interface VirtualMachinesLoginsItems {
    hostname: string;
    username: string;
    public_key: string;
    userUuid: string;
    virtualMachineUuid: string;
    authorizedKeyUuid: string;
}

export interface VirtualMachineLogins {
    kind: string;
    items: VirtualMachinesLoginsItems[];
}