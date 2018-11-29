// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "~/models/resource";

export interface VirtualMachinesResource extends Resource {
    hostname: string;
}

export interface VirtualMachinesLoginsResource {
    hostname: string;
    username: string;
    public_key: string;
    user_uuid: string;
    virtual_machine_uuid: string;
    authorized_key_uuid: string;
}