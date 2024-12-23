// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, CopyIcon, RenameIcon, ShareIcon, FileCopyOutlinedIcon } from "components/icon/icon";
import { openMoveCollectionDialog } from "store/collections/collection-move-actions";
import { openCollectionCopyDialog, openMultiCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { msCommonActionSet, MultiSelectMenuActionSet, MultiSelectMenuAction } from "./ms-menu-actions";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { TrashIcon, Link, FolderSharedIcon } from "components/icon/icon";
import { openCollectionUpdateDialog } from "store/collections/collection-update-actions";
import { copyToClipboardAction, copyStringToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { openWebDavS3InfoDialog } from "store/collections/collection-info-actions";
import { openSharingDialog } from "store/sharing-dialog/sharing-dialog-actions";


const { MAKE_A_COPY, MOVE_TO, MOVE_TO_TRASH, EDIT_COLLECTION, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, COPY_LINK_TO_CLIPBOARD, COPY_UUID, VIEW_DETAILS, API_DETAILS, ADD_TO_FAVORITES, SHARE} = ContextMenuActionNames;

const msCopyCollection: MultiSelectMenuAction = {
    name: MAKE_A_COPY,
    icon: FileCopyOutlinedIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, [...resources]) => {
        if (resources[0].fromContextMenu || resources.length === 1) dispatch<any>(openCollectionCopyDialog(resources[0]));
        else dispatch<any>(openMultiCollectionCopyDialog(resources[0]));
    },
}

const msMoveCollection: MultiSelectMenuAction = {
    name: MOVE_TO,
    icon: MoveToIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => dispatch<any>(openMoveCollectionDialog(resources[0])),
}

const msToggleTrashAction: MultiSelectMenuAction = {
    name: MOVE_TO_TRASH,
    icon: TrashIcon,
    isForMulti: true,
    hasAlts: false,
    execute: (dispatch, resources: ContextMenuResource[]) => {
        for (const resource of [...resources]) {
            dispatch<any>(toggleCollectionTrashed(resource.uuid, resource.isTrashed!!));
        }
    },
}

const msEditCollection: MultiSelectMenuAction = {
    name: ContextMenuActionNames.EDIT_COLLECTION,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openCollectionUpdateDialog(resources[0]));
    },
}

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_LINK_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

const msCopyUuid: MultiSelectMenuAction  = {
    name: COPY_UUID,
    icon: CopyIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
    },
};

const msOpenWith3rdPartyClientAction: MultiSelectMenuAction  = {
    name: OPEN_WITH_3RD_PARTY_CLIENT,
    icon: FolderSharedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

const msShareAction: MultiSelectMenuAction  = {
    name: SHARE,
    icon: ShareIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openSharingDialog(resources[0].uuid));
    },
};

export const msCollectionActionSet: MultiSelectMenuActionSet = [
    [
        ...msCommonActionSet,
        msCopyCollection,
        msMoveCollection,
        msToggleTrashAction,
        msEditCollection,
        msCopyToClipboardMenuAction,
        msOpenWith3rdPartyClientAction,
        msShareAction,
        msCopyUuid,
    ],
];

export const msReadOnlyCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, ADD_TO_FAVORITES, OPEN_WITH_3RD_PARTY_CLIENT]);
export const msCommonCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID,MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, OPEN_WITH_3RD_PARTY_CLIENT, EDIT_COLLECTION, SHARE, MOVE_TO, ADD_TO_FAVORITES, MOVE_TO_TRASH])
export const msOldCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, OPEN_WITH_3RD_PARTY_CLIENT, EDIT_COLLECTION, SHARE, MOVE_TO, ADD_TO_FAVORITES, MOVE_TO_TRASH])
export const msWriteableCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, OPEN_WITH_3RD_PARTY_CLIENT, EDIT_COLLECTION, MOVE_TO, ADD_TO_FAVORITES, MOVE_TO_TRASH])