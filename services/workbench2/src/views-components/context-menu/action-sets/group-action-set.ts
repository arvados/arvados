// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { RenameIcon, AdvancedIcon, RemoveIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openRemoveGroupDialog, openGroupUpdateDialog } from 'store/groups-panel/groups-panel-actions';

export const groupActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.RENAME,
            icon: RenameIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openGroupUpdateDialog(resources[0]))
            },
        },
        {
            name: ContextMenuActionNames.API_DETAILS,
            icon: AdvancedIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.REMOVE,
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveGroupDialog(resources[0].uuid));
            },
        },
    ],
];
