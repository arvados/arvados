// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction, ContextMenuActionSet, ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openMultipleFilesRemoveDialog } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import {
    openCollectionPartialCopyMultipleToNewCollectionDialog,
    openCollectionPartialCopyMultipleToExistingCollectionDialog,
    openCollectionPartialCopyToSeparateCollectionsDialog
} from 'store/collections/collection-partial-copy-actions';
import { openCollectionPartialMoveMultipleToExistingCollectionDialog, openCollectionPartialMoveMultipleToNewCollectionDialog, openCollectionPartialMoveToSeparateCollectionsDialog } from "store/collections/collection-partial-move-actions";
import { FileCopyIcon, FileMoveIcon, RemoveIcon, SelectAllIcon, SelectNoneIcon } from "components/icon/icon";

const copyActions: ContextMenuAction[] = [
    {
        name: ContextMenuActionNames.COPY_SELECTED_INTO_NEW_COLLECTION,
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyMultipleToNewCollectionDialog());
        }
    },
    {
        name: ContextMenuActionNames.COPY_SELECTED_INTO_EXISTING_COLLECTION,
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyMultipleToExistingCollectionDialog());
        }
    },
];

const copyActionsMultiple: ContextMenuAction[] = [
    ...copyActions,
    {
        name: ContextMenuActionNames.COPY_SELECTED_INTO_SEPARATE_COLLECTIONS,
        icon: FileCopyIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyToSeparateCollectionsDialog());
        }
    }
];

const moveActions: ContextMenuAction[] = [
    {
        name: ContextMenuActionNames.MOVE_SELECTED_INTO_NEW_COLLECTION,
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveMultipleToNewCollectionDialog());
        }
    },
    {
        name: ContextMenuActionNames.MOVE_SELECTED_INTO_EXISTING_COLLECTION,
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveMultipleToExistingCollectionDialog());
        }
    },
];

const moveActionsMultiple: ContextMenuAction[] = [
    ...moveActions,
    {
        name: ContextMenuActionNames.MOVE_SELECTED_INTO_SEPARATE_COLLECTIONS,
        icon: FileMoveIcon,
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveToSeparateCollectionsDialog());
        }
    }
];

const selectActions: ContextMenuAction[] = [
    {
        name: ContextMenuActionNames.SELECT_ALL,
        icon: SelectAllIcon,
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
        }
    },
    {
        name: ContextMenuActionNames.UNSELECT_ALL,
        icon: SelectNoneIcon,
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.UNSELECT_ALL_COLLECTION_FILES());
        }
    },
];

const removeAction: ContextMenuAction = {
    name: ContextMenuActionNames.REMOVE_SELECTED,
    icon: RemoveIcon,
    execute: dispatch => {
        dispatch(openMultipleFilesRemoveDialog());
    }
};

// These action sets are used on the multi-select actions button.
export const readOnlyCollectionFilesActionSet: ContextMenuActionSet = [
    selectActions,
    copyActions,
];

export const readOnlyCollectionFilesMultipleActionSet: ContextMenuActionSet = [
    selectActions,
    copyActionsMultiple,
];

export const collectionFilesActionSet: ContextMenuActionSet = readOnlyCollectionFilesActionSet.concat([[
    removeAction,
    ...moveActions
]]);

export const collectionFilesMultipleActionSet: ContextMenuActionSet = readOnlyCollectionFilesMultipleActionSet.concat([[
    removeAction,
    ...moveActionsMultiple
]]);
