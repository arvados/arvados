// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "../context-menu-action-set";
import { FileCopyIcon, FileMoveIcon, RemoveIcon, RenameIcon } from "components/icon/icon";
import { DownloadCollectionFileAction } from "../actions/download-collection-file-action";
import { openFileRemoveDialog, openRenameFileDialog } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { CollectionFileViewerAction } from "views-components/context-menu/actions/collection-file-viewer-action";
import { CollectionCopyToClipboardAction } from "../actions/collection-copy-to-clipboard-action";
import {
    openCollectionPartialMoveToExistingCollectionDialog,
    openCollectionPartialMoveToNewCollectionDialog,
} from "store/collections/collection-partial-move-actions";
import {
    openCollectionPartialCopyToExistingCollectionDialog,
    openCollectionPartialCopyToNewCollectionDialog,
} from "store/collections/collection-partial-copy-actions";

export const readOnlyCollectionDirectoryItemActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.COPY_ITEM_INTO_NEW_COLLECTION,
            icon: FileCopyIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialCopyToNewCollectionDialog(resources[0]));
            },
        },
        {
            name: ContextMenuActionNames.COPY_ITEM_INTO_EXISTING_COLLECTION,
            icon: FileCopyIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialCopyToExistingCollectionDialog(resources[0]));
            },
        },
        {
            component: CollectionFileViewerAction,
            name: ContextMenuActionNames.OPEN_IN_NEW_TAB,
            execute: () => {
                return;
            },
        },
        {
            component: CollectionCopyToClipboardAction,
            name: ContextMenuActionNames.COPY_TO_CLIPBOARD,
            execute: () => {
                return;
            },
        },
    ],
];

export const readOnlyCollectionFileItemActionSet: ContextMenuActionSet = [
    [
        {
            component: DownloadCollectionFileAction,
            name: ContextMenuActionNames.DOWNLOAD,
            execute: () => {
                return;
            },
        },
        ...readOnlyCollectionDirectoryItemActionSet.reduce((prev, next) => prev.concat(next), []),
    ],
];

const writableActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.MOVE_ITEM_INTO_NEW_COLLECTION,
            icon: FileMoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialMoveToNewCollectionDialog(resources[0]));
            },
        },
        {
            name: ContextMenuActionNames.MOVE_ITEM_INTO_EXISTING_COLLECTION,
            icon: FileMoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialMoveToExistingCollectionDialog(resources[0]));
            },
        },
        {
            name: ContextMenuActionNames.RENAME,
            icon: RenameIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(
                    openRenameFileDialog({
                        name: resources[0].name,
                        id: resources[0].uuid,
                        path: resources[0].uuid.split("/").slice(1).join("/"),
                    })
                );
            },
        },
        {
            name: ContextMenuActionNames.REMOVE,
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openFileRemoveDialog(resources[0].uuid));
            },
        },
    ],
];

export const collectionDirectoryItemActionSet: ContextMenuActionSet = readOnlyCollectionDirectoryItemActionSet.concat(writableActionSet);

export const collectionFileItemActionSet: ContextMenuActionSet = readOnlyCollectionFileItemActionSet.concat(writableActionSet);
