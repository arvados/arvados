// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { RenameIcon, DownloadIcon, RemoveIcon } from "~/components/icon/icon";
import { openRenameFileDialog } from "../../rename-file-dialog/rename-file-dialog";
import { DownloadCollectionFileAction } from "../actions/download-collection-file-action";
import { openFileRemoveDialog } from "../../../store/collection-panel/collection-panel-files/collection-panel-files-actions";


export const collectionFilesItemActionSet: ContextMenuActionSet = [[{
    name: "Rename",
    icon: RenameIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRenameFileDialog(resource.name));
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
}]];
