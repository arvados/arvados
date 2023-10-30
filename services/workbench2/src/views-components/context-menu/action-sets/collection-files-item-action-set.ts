// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
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
            name: "Copy item into new collection",
            icon: FileCopyIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialCopyToNewCollectionDialog(resources[0]));
            },
        },
        {
            name: "Copy item into existing collection",
            icon: FileCopyIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialCopyToExistingCollectionDialog(resources[0]));
            },
        },
        {
            component: CollectionFileViewerAction,
            execute: () => {
                return;
            },
        },
        {
            component: CollectionCopyToClipboardAction,
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
            name: "Move item into new collection",
            icon: FileMoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialMoveToNewCollectionDialog(resources[0]));
            },
        },
        {
            name: "Move item into existing collection",
            icon: FileMoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openCollectionPartialMoveToExistingCollectionDialog(resources[0]));
            },
        },
        {
            name: "Rename",
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
            name: "Remove",
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openFileRemoveDialog(resources[0].uuid));
            },
        },
    ],
];

export const collectionDirectoryItemActionSet: ContextMenuActionSet = readOnlyCollectionDirectoryItemActionSet.concat(writableActionSet);

export const collectionFileItemActionSet: ContextMenuActionSet = readOnlyCollectionFileItemActionSet.concat(writableActionSet);
