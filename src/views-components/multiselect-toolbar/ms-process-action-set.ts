// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, RemoveIcon, ReRunProcessIcon } from "components/icon/icon";
import { openMoveProcessDialog } from "store/processes/process-move-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from "./ms-menu-action-set";

export const msProcessActionSet: MultiSelectMenuActionSet = [
    [
        {
            icon: ReRunProcessIcon,
            name: MultiSelectMenuActionNames.COPY_AND_RERUN_PROCESS,
            isForMulti: true,
            execute: (dispatch, resources) => {
                for (const resource of [...resources]) {
                    dispatch<any>(openCopyProcessDialog(resource));
                }
            },
        },
        {
            icon: MoveToIcon,
            name: MultiSelectMenuActionNames.MOVE_TO,
            isForMulti: true,
            execute: (dispatch, resources) => {
                dispatch<any>(openMoveProcessDialog(resources[0]));
            },
        },
        {
            name: MultiSelectMenuActionNames.REMOVE,
            icon: RemoveIcon,
            isForMulti: true,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
            },
        },
    ],
];
