// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from "views-components/context-menu/context-menu-action-set";
import { collectionPanelFilesAction } from "store/collection-panel/collection-panel-files/collection-panel-files-actions";
import { SelectAllIcon } from "components/icon/icon";

export const collectionFilesNotSelectedActionSet: ContextMenuActionItemSet = [[{
    name: "Select all",
    icon: SelectAllIcon,
    execute: dispatch => {
        dispatch(collectionPanelFilesAction.SELECT_ALL_COLLECTION_FILES());
    }
}]];
