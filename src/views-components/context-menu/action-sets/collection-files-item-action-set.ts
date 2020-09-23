// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { RemoveIcon, RenameIcon } from "~/components/icon/icon";
import { DownloadCollectionFileAction } from "../actions/download-collection-file-action";
import { openFileRemoveDialog, openRenameFileDialog } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { CollectionFileViewerAction } from '~/views-components/context-menu/actions/collection-file-viewer-action';


export const readOnlyCollectionFilesItemActionSet: ContextMenuActionSet = [[
    {
        component: DownloadCollectionFileAction,
        execute: () => { return; }
    },
    {
        component: CollectionFileViewerAction,
        execute: () => { return; },
    }
]];

export const collectionFilesItemActionSet: ContextMenuActionSet = readOnlyCollectionFilesItemActionSet.concat([[
    {
        name: "Rename",
        icon: RenameIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openRenameFileDialog({ name: resource.name, id: resource.uuid }));
        }
    },
    {
        name: "Remove",
        icon: RemoveIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openFileRemoveDialog(resource.uuid));
        }
    }
]]);