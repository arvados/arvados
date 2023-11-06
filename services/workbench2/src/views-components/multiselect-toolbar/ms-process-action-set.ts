// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { MoveToIcon, RemoveIcon, ReRunProcessIcon } from "components/icon/icon";
import { openMoveProcessDialog } from "store/processes/process-move-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";

export const msProcessActionSet: ContextMenuActionSet = [
    [
        {
            icon: ReRunProcessIcon,
            name: "Copy and re-run process",
            execute: (dispatch, resources) => {
                for (const resource of [...resources]) {
                    dispatch<any>(openCopyProcessDialog(resource));
                }
            },
        },
        {
            icon: MoveToIcon,
            name: "Move to",
            execute: (dispatch, resources) => {
                dispatch<any>(openMoveProcessDialog(resources[0]));
            },
        },
        {
            name: "Remove",
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
            },
        },
    ],
];
