// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { RenameIcon, AdvancedIcon, RemoveIcon, AttributesIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openGroupAttributes, openRemoveGroupDialog, openGroupUpdateDialog } from 'store/groups-panel/groups-panel-actions';

export const groupActionSet: ContextMenuActionSet = [
    [
        {
            name: 'Rename',
            icon: RenameIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openGroupUpdateDialog(resources[0]))
            },
        },
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openGroupAttributes(resources[0].uuid))
            },
        },
        {
            name: 'API Details',
            icon: AdvancedIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: 'Remove',
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveGroupDialog(resources[0].uuid));
            },
        },
    ],
];
