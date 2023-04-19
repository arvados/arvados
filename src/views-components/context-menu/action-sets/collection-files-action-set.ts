// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openMultipleFilesRemoveDialog } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import {
    openCollectionPartialCopyToNewCollectionDialog,
    openCollectionPartialCopyToExistingCollectionDialog,
    openCollectionPartialCopyToSeparateCollectionsDialog
} from 'store/collections/collection-partial-copy-actions';
import { openCollectionPartialMoveToExistingCollectionDialog, openCollectionPartialMoveToNewCollectionDialog, openCollectionPartialMoveToSeparateCollectionsDialog } from "store/collections/collection-partial-move-actions";

// These action sets are used on the multi-select actions button.
export const readOnlyCollectionFilesActionSet: ContextMenuActionSet = [[
    {
        name: "Select all",
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
        }
    },
    {
        name: "Unselect all",
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.UNSELECT_ALL_COLLECTION_FILES());
        }
    },
    {
        name: "Copy selected into new collection",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyToNewCollectionDialog());
        }
    },
    {
        name: "Copy selected into existing collection",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyToExistingCollectionDialog());
        }
    },
    {
        name: "Copy selected into separate collections",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialCopyToSeparateCollectionsDialog());
        }
    }
]];

export const collectionFilesActionSet: ContextMenuActionSet = readOnlyCollectionFilesActionSet.concat([[
    {
        name: "Remove selected",
        execute: dispatch => {
            dispatch(openMultipleFilesRemoveDialog());
        }
    },
    {
        name: "Move selected into new collection",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveToNewCollectionDialog());
        }
    },
    {
        name: "Move selected into existing collection",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveToExistingCollectionDialog());
        }
    },
    {
        name: "Move selected into separate collections",
        execute: dispatch => {
            dispatch<any>(openCollectionPartialMoveToSeparateCollectionsDialog());
        }
    }
]]);
