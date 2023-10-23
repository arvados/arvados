// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { CanReadIcon, CanManageIcon, CanWriteIcon } from "components/icon/icon";
import { editPermissionLevel } from 'store/group-details-panel/group-details-panel-actions';
import { PermissionLevel } from "models/permission";

export const permissionEditActionSet: ContextMenuActionSet = [[{
    name: "Read",
    icon: CanReadIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(editPermissionLevel(uuid, PermissionLevel.CAN_READ));
    }
}, {
    name: "Write",
    icon: CanWriteIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(editPermissionLevel(uuid, PermissionLevel.CAN_WRITE));
    }
}, {
    name: "Manage",
    icon: CanManageIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(editPermissionLevel(uuid, PermissionLevel.CAN_MANAGE));
    }
}]];
