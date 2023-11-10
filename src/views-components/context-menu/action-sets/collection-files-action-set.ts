// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItem, ContextMenuActionItemSet } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openMultipleFilesRemoveDialog } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import {
    openCollectionPartialCopyMultipleToNewCollectionDialog,
    openCollectionPartialCopyMultipleToExistingCollectionDialog,
    openCollectionPartialCopyToSeparateCollectionsDialog
} from 'store/collections/collection-partial-copy-actions';
import { openCollectionPartialMoveMultipleToExistingCollectionDialog, openCollectionPartialMoveMultipleToNewCollectionDialog, openCollectionPartialMoveToSeparateCollectionsDialog } from "store/collections/collection-partial-move-actions";
import { FileCopyIcon, FileMoveIcon, RemoveIcon, SelectAllIcon, SelectNoneIcon } from "components/icon/icon";

const copyActions: ContextMenuActionItem[] = [
    {
        name: "Copy selected into new collection",
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyMultipleToNewCollectionDialog());
        }
    },
    {
        name: "Copy selected into existing collection",
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyMultipleToExistingCollectionDialog());
        }
    },
];

const copyActionsMultiple: ContextMenuActionItem[] = [
    ...copyActions,
    {
        name: "Copy selected into separate collections",
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyToSeparateCollectionsDialog());
        }
    }
];

const moveActions: ContextMenuActionItem[] = [
    {
        name: "Move selected into new collection",
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveMultipleToNewCollectionDialog());
        }
    },
    {
        name: "Move selected into existing collection",
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveMultipleToExistingCollectionDialog());
        }
    },
];

const moveActionsMultiple: ContextMenuActionItem[] = [
    ...moveActions,
    {
        name: "Move selected into separate collections",
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveToSeparateCollectionsDialog());
        }
    }
];

const selectActions: ContextMenuActionItem[] = [
    {
        name: "Select all",
        icon: SelectAllIcon,
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
        }
    },
    {
        name: "Unselect all",
        icon: SelectNoneIcon,
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.UNSELECT_ALL_COLLECTION_FILES());
        }
    },
];

const removeAction: ContextMenuActionItem = {
    name: "Remove selected",
    icon: RemoveIcon,
    execute: dispatch => {
        dispatch(openMultipleFilesRemoveDialog());
    }
};

// These action sets are used on the multi-select actions button.
export const readOnlyCollectionFilesActionSet: ContextMenuActionItemSet = [
    selectActions,
    copyActions,
];

export const readOnlyCollectionFilesMultipleActionSet: ContextMenuActionItemSet = [
    selectActions,
    copyActionsMultiple,
];

export const collectionFilesActionSet: ContextMenuActionItemSet = readOnlyCollectionFilesActionSet.concat([[
    removeAction,
    ...moveActions
]]);

export const collectionFilesMultipleActionSet: ContextMenuActionItemSet = readOnlyCollectionFilesMultipleActionSet.concat([[
    removeAction,
    ...moveActionsMultiple
]]);
