// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { RenameIcon, AdvancedIcon, DeleteForever } from "components/icon/icon";
import { openProjectUpdateDialog } from "store/projects/project-update-actions";
import { ShareIcon } from "components/icon/icon";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";

export const advancedAction = {
    icon: AdvancedIcon,
    name: ContextMenuActionNames.API_DETAILS,
    execute: (dispatch, resources) => {
        dispatch(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const editExternalCredentialAction = {
    icon: RenameIcon,
    name: ContextMenuActionNames.EDIT_CREDENTIAL,
    execute: (dispatch, resources) => {
        // dispatch(openProjectUpdateDialog(resources[0]));
    },
};

export const shareAction = {
    icon: ShareIcon,
    name: ContextMenuActionNames.SHARE,
    execute: (dispatch, resources) => {
        dispatch(openSharingDialog(resources[0].uuid));
    },
};

export const deleteAction = {
    name: ContextMenuActionNames.REMOVE,
    icon: DeleteForever,
    isForMulti: true,
    execute: (dispatch, resources) => {
        // dispatch<any>(openRemoveProcessDialog(resources[0], resources.length));
    },
};


export const externalCredentialActionSet: ContextMenuActionSet = [
    [
        advancedAction,
        editExternalCredentialAction,
        shareAction,
        deleteAction,
    ],
];
