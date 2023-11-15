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
            name: MultiSelectMenuActionNames.COPY_AND_RERUN_PROCESS,
            icon: ReRunProcessIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                for (const resource of [...resources]) {
                    dispatch<any>(openCopyProcessDialog(resource));
                }
            },
        },
        {
            name: MultiSelectMenuActionNames.MOVE_TO,
            icon: MoveToIcon,
            hasAlts: false,
            isForMulti: true,
            execute: (dispatch, resources) => {
                dispatch<any>(openMoveProcessDialog(resources[0]));
            },
        },
        {
            name: MultiSelectMenuActionNames.REMOVE,
            icon: RemoveIcon,
            hasAlts: false,
            isForMulti: true,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
            },
        },
    ],
];
