// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, ShareIcon } from "~/components/icon/icon";
import { openFileRemoveDialog, openRenameFileDialog } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';

export const repositoryActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AdvancedIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openRenameFileDialog({ name: resource.name, id: resource.uuid }));
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
        dispatch<any>(openFileRemoveDialog(resource.uuid));
    }
},
{
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openFileRemoveDialog(resource.uuid));
    }
}]];
