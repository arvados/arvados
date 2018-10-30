// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openMultipleFilesRemoveDialog } from "~/store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { openCollectionPartialCopyDialog } from '~/store/collections/collection-partial-copy-actions';

export const collectionFilesActionSet: ContextMenuActionSet = [[{
    name: "Select all",
    execute: dispatch => {
        dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
    }
}, {
    name: "Unselect all",
    execute: dispatch => {
        dispatch(collectionPanelFilesAction.UNSELECT_ALL_COLLECTION_FILES());
    }
}, {
    name: "Remove selected",
    execute: dispatch => {
        dispatch(openMultipleFilesRemoveDialog());
    }
}, {
    name: "Download selected",
    execute: () => { return; }
}, {
    name: "Create a new collection with selected",
    execute: dispatch => { 
        dispatch<any>(openCollectionPartialCopyDialog());
    }
}]];
