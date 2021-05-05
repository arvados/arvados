// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { RemoveIcon, RenameIcon } from "~/components/icon/icon";
import { DownloadCollectionFileAction } from "../actions/download-collection-file-action";
import { openFileRemoveDialog, openRenameFileDialog } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { CollectionFileViewerAction } from '~/views-components/context-menu/actions/collection-file-viewer-action';
import { CollectionCopyToClipboardAction } from "../actions/collection-copy-to-clipboard-action";

export const readOnlyCollectionDirectoryItemActionSet: ContextMenuActionSet = [[
    {
        component: CollectionFileViewerAction,
        execute: () => { return; },
    },
    {
        component: CollectionCopyToClipboardAction,
        execute: () => { return; },
    }
]];

export const readOnlyCollectionFileItemActionSet: ContextMenuActionSet = [[
    {
        component: DownloadCollectionFileAction,
        execute: () => { return; }
    },
    ...readOnlyCollectionDirectoryItemActionSet.reduce((prev, next) => prev.concat(next), []),
]];

const writableActionSet: ContextMenuActionSet = [[
    {
        name: "Rename",
        icon: RenameIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openRenameFileDialog({
                name: resource.name,
                id: resource.uuid,
                path: resource.uuid.split('/').slice(1).join('/') }));
        }
    },
    {
        name: "Remove",
        icon: RemoveIcon,
        execute: (dispatch, resource) => {
            dispatch<any>(openFileRemoveDialog(resource.uuid));
        }
    }
]];

export const collectionDirectoryItemActionSet: ContextMenuActionSet = readOnlyCollectionDirectoryItemActionSet.concat(writableActionSet);

export const collectionFileItemActionSet: ContextMenuActionSet = readOnlyCollectionFileItemActionSet.concat(writableActionSet);