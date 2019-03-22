// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openMultipleFilesRemoveDialog } from "~/store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { openCollectionPartialCopyDialog, openCollectionPartialCopyToSelectedCollectionDialog } from '~/store/collections/collection-partial-copy-actions';
import { DownloadCollectionFileAction } from "~/views-components/context-menu/actions/download-collection-file-action";

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
    component: DownloadCollectionFileAction,
    execute: () => { return; }
}, {
    name: "Create a new collection with selected",
    execute: dispatch => {
        dispatch<any>(openCollectionPartialCopyDialog());
    }
}, {
    name: "Copy selected into the collection",
    execute: dispatch => {
        dispatch<any>(openCollectionPartialCopyToSelectedCollectionDialog());
    }
}]];
