// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { RenameIcon, AdvancedIcon, DeleteForever, CopyIcon } from "components/icon/icon";
import { openRemoveExternalCredentialDialog, openExternalCredentialUpdateDialog } from "store/external-credentials/external-credentials-actions";
import { ShareIcon } from "components/icon/icon";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";
import { copyStringToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";

export const advancedAction = {
    icon: AdvancedIcon,
    name: ContextMenuActionNames.API_DETAILS,
    execute: (dispatch: Dispatch, resources: ContextMenuResource[]) => {
        dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const editExternalCredentialAction = {
    icon: RenameIcon,
    name: ContextMenuActionNames.EDIT_CREDENTIAL,
    execute: (dispatch: Dispatch, resources: ContextMenuResource[]) => {
        dispatch<any>(openExternalCredentialUpdateDialog(resources[0]));
    },
};

export const shareAction = {
    icon: ShareIcon,
    name: ContextMenuActionNames.SHARE,
    execute: (dispatch: Dispatch, resources: ContextMenuResource[]) => {
        dispatch<any>(openSharingDialog(resources[0].uuid));
    },
};

export const deleteAction = {
    name: ContextMenuActionNames.REMOVE,
    icon: DeleteForever,
    isForMulti: true,
    execute: (dispatch: Dispatch, resources: ContextMenuResource[]) => {
        dispatch<any>(openRemoveExternalCredentialDialog(resources[0]));
    },
};

export const copyUuidAction = {
    icon: CopyIcon,
    name: ContextMenuActionNames.COPY_UUID,
    execute: (dispatch: Dispatch, resources: ContextMenuResource[]) => {
        dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
    },
};

export const externalCredentialActionSet: ContextMenuActionSet = [
    [
        advancedAction,
        editExternalCredentialAction,
        shareAction,
        deleteAction,
        copyUuidAction,
    ],
];
