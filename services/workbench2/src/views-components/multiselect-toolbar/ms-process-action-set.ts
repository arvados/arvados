// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, DeleteForever, ReRunProcessIcon, OutputIcon, RenameIcon, StopIcon } from "components/icon/icon";
import { openMoveProcessDialog } from "store/processes/process-move-actions";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog } from "store/processes/processes-actions";
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from "./ms-menu-actions";
import { MultiSelectMenuActionNames } from "components/multiselect-toolbar/ms-menu-actions"; 
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { msNavigateToOutput } from "store/multiselect/multiselect-actions";
import { cancelRunningWorkflow } from "store/processes/processes-actions";

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
    icon: DeleteForever,
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

const msCancelProcess: MultiSelectMenuAction = {
    name: MultiSelectMenuActionNames.CANCEL,
    icon: StopIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(cancelRunningWorkflow(resources[0].uuid));
    },
}

export const msProcessActionSet: MultiSelectMenuActionSet = [
    [
        ...msCommonActionSet,
        msCopyAndRerunProcess,
        msRemoveProcess,
        msMoveTo,
        msViewOutputs,
        msEditProcess,
        msCancelProcess
    ]
];

const { MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS, CANCEL } = MultiSelectMenuActionNames

export const msCommonProcessActionFilter = new Set([MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, OUTPUTS, EDIT_PROCESS ]);
export const msRunningProcessActionFilter = new Set([MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, OUTPUTS, EDIT_PROCESS, CANCEL ]);

export const msReadOnlyProcessActionFilter = new Set([COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, OUTPUTS ]);
export const msAdminProcessActionFilter = new Set([MOVE_TO, REMOVE, COPY_AND_RERUN_PROCESS, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS ]);

