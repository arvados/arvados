// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, ShareIcon, AttributesIcon } from "~/components/icon/icon";
import { openRenameFileDialog } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { openRepositoryAttributes, openRemoveRepositoryDialog } from "~/store/repositories/repositories-actions";

export const repositoryActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRepositoryAttributes(resource.index!));
    }
}, {
    name: "Share",
    icon: ShareIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRenameFileDialog({ name: resource.name, id: resource.uuid }));
    }
}, {
    name: "Advanced",
    icon: AdvancedIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openAdvancedTabDialog(resource.uuid, resource.index));
    }
}, {
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRemoveRepositoryDialog(resource.uuid));
    }
}]];
