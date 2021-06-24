// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";

export const collectionFilesNotSelectedActionSet: ContextMenuActionSet = [[{
    name: "Select all",
    execute: dispatch => {
        dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
    }
}]];