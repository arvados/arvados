// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { RenameIcon, RemoveIcon } from "~/components/icon/icon";
import { DownloadCollectionFileAction } from "../actions/download-collection-file-action";
import { openFileRemoveDialog, openRenameFileDialog } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { FileViewerActions } from '~/views-components/context-menu/actions/file-viewer-actions';


export const collectionFilesItemActionSet: ContextMenuActionSet = [[{
    name: "Rename",
    icon: RenameIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRenameFileDialog({ name: resource.name, id: resource.uuid }));
    }
}, {
    component: DownloadCollectionFileAction,
    execute: () => { return; }
}, {
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openFileRemoveDialog(resource.uuid));
    }
}], [{
    component: FileViewerActions,
    execute: () => { return; },
}]];
