// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { DeleteForever, ReRunProcessIcon, OutputIcon, RenameIcon, StopIcon, CopyIcon, Link } from "components/icon/icon";
import { openCopyProcessDialog } from "store/processes/process-copy-actions";
import { openRemoveProcessDialog, cancelRunningWorkflow } from "store/processes/processes-actions";
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from "./ms-menu-actions";
import { openProcessUpdateDialog } from "store/processes/process-update-actions";
import { msNavigateToOutput } from "store/multiselect/multiselect-actions";
import { copyToClipboardAction, copyStringToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";

const {REMOVE, COPY_AND_RERUN_PROCESS, COPY_LINK_TO_CLIPBOARD, COPY_UUID, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS, CANCEL } = ContextMenuActionNames

const msCopyAndRerunProcess: MultiSelectMenuAction = {
    name: COPY_AND_RERUN_PROCESS,
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
    name: REMOVE,
    icon: DeleteForever,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => {
        dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
    },
}

const msCopyUuid: MultiSelectMenuAction  = {
    name: COPY_UUID,
    icon: CopyIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
    },
};

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_LINK_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

// removed until auto-move children is implemented
// const msMoveTo: MultiSelectMenuAction = {
//     name: MOVE_TO,
//     icon: MoveToIcon,
//     hasAlts: false,
//     isForMulti: true,
//     execute: (dispatch, resources) => {
//         dispatch<any>(openMoveProcessDialog(resources[0]));
//     },
// }

const msViewOutputs: MultiSelectMenuAction = {
    name: OUTPUTS,
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
    name: EDIT_PROCESS,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openProcessUpdateDialog(resources[0]));
    },
}

const msCancelProcess: MultiSelectMenuAction = {
    name: CANCEL,
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
        // msMoveTo,
        msViewOutputs,
        msEditProcess,
        msCancelProcess,
        msCopyUuid,
        msCopyToClipboardMenuAction,
    ]
];

export const msCommonProcessActionFilter = new Set([REMOVE, COPY_AND_RERUN_PROCESS, COPY_LINK_TO_CLIPBOARD, COPY_UUID, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, OUTPUTS, EDIT_PROCESS ]);
export const msRunningProcessActionFilter = new Set([REMOVE, COPY_AND_RERUN_PROCESS, COPY_LINK_TO_CLIPBOARD, COPY_UUID, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, OUTPUTS, EDIT_PROCESS, CANCEL ]);

export const msReadOnlyProcessActionFilter = new Set([COPY_AND_RERUN_PROCESS, COPY_LINK_TO_CLIPBOARD, COPY_UUID, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, OUTPUTS ]);
export const msAdminProcessActionFilter = new Set([REMOVE, COPY_AND_RERUN_PROCESS, COPY_LINK_TO_CLIPBOARD, COPY_UUID, ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, VIEW_DETAILS, API_DETAILS, SHARE, ADD_TO_PUBLIC_FAVORITES, OUTPUTS, EDIT_PROCESS ]);
