// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource, LinkClass } from './link';

export interface PermissionResource extends LinkResource {
    linkClass: LinkClass.PERMISSION;
}

export enum PermissionLevel {
    NONE = 'none',
    CAN_READ = 'can_read',
    CAN_WRITE = 'can_write',
    CAN_MANAGE = 'can_manage',
    CAN_LOGIN = 'can_login',
}
