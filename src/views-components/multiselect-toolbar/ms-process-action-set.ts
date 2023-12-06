// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, RemoveIcon, ReRunProcessIcon, OutputIcon, RenameIcon } from "components/icon/icon";
import { openMoveProcessDialog } from "store/processes/process-move-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from "./ms-menu-actions";
import { MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-actions";
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { msNavigateToOutput } from "store/multiselect/multiselect-actions";

const msCopyAndRerunProcess: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.COPY_AND_RERUN_PROCESS,
    icon: ReRunProcessIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch<any>(openCopyProcessDialog(resource));
        }
    },
}

const msRemoveProcess: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.REMOVE,
    icon: RemoveIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => {
        dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
    },
}

const msMoveTo: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.MOVE_TO,
    icon: MoveToIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => {
        dispatch<any>(openMoveProcessDialog(resources[0]));
    },
}

const msViewOutputs: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.OUTPUTS,
    icon: OutputIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
                if (resources[0]) {
            dispatch<any>(msNavigateToOutput(resources[0]));
        }
    },
}

const msEditProcess: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.EDIT_PROCESS,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openProcessUpdateDialog(resources[0]));
    },
}

export const msProcessActionSet: MultiSelectMenuActionSet = [
    [
        ...msCommonActionSet,
        msCopyAndRerunProcess,
        msRemoveProcess,
        msMoveTo,
        msViewOutputs,
        msEditProcess
    ]
];

const { MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS } = MultiSelectMenuActionNames

export const msCommonProcessActionFilter = new Set([MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS ]);
