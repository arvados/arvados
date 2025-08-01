// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction, openDownloadZipDialog } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { DownloadIcon, SelectAllIcon } from "components/icon/icon";

export const collectionFilesNotSelectedActionSet: ContextMenuActionSet = [
    [{
        name: ContextMenuActionNames.SELECT_ALL,
        icon: SelectAllIcon,
        execute: dispatch => {
            dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
        }
    }],
    [{
        name: ContextMenuActionNames.DOWNLOAD_ALL,
        icon: DownloadIcon,
        execute: dispatch => {
            dispatch<any>(openDownloadZipDialog());
        }
    }],
];
