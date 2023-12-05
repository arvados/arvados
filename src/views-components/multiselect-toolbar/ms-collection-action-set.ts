// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MoveToIcon, CopyIcon, RenameIcon } from "components/icon/icon";
import { openMoveCollectionDialog } from "store/collections/collection-move-actions";
import { openCollectionCopyDialog, openMultiCollectionCopyDialog } from "store/collections/collection-copy-actions";
import { toggleCollectionTrashed } from "store/trash/trash-actions";
import { ContextMenuResource } from "store/context-menu/context-menu-actions";
import { msCommonActionSet, MultiSelectMenuActionSet, MultiSelectMenuActionNames, MultiSelectMenuAction } from "./ms-menu-actions";
import { TrashIcon } from "components/icon/icon";
import { openCollectionUpdateDialog } from "store/collections/collection-update-actions";

const { MAKE_A_COPY, MOVE_TO, ADD_TO_TRASH, EDIT_COLLECTION, OPEN_IN_NEW_TAB, OPEN_W_3RD_PARTY_CLIENT, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, ADD_TO_FAVORITES, SHARE} = MultiSelectMenuActionNames;

const msCopyCollection: MultiSelectMenuAction = {
    name: MAKE_A_COPY,
    icon: CopyIcon,
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
    name: ADD_TO_TRASH,
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
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    name: "Edit collection",
    execute: (dispatch, resources) => {
        dispatch<any>(openCollectionUpdateDialog(resources[0]));
    },
}

export const msCollectionActionSet: MultiSelectMenuActionSet = [
    [
        ...msCommonActionSet,
        msCopyCollection,
        msMoveCollection,
        msToggleTrashAction,
        msEditCollection
    ],
];

export const msReadOnlyCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, ADD_TO_FAVORITES, OPEN_W_3RD_PARTY_CLIENT]);
export const msCommonCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT, EDIT_COLLECTION, SHARE, MOVE_TO, ADD_TO_TRASH])
export const msOldCollectionActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, MAKE_A_COPY, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT, EDIT_COLLECTION, SHARE, MOVE_TO, ADD_TO_TRASH])