// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, AttributesIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openGroupMemberAttributes, openRemoveGroupMemberDialog } from 'store/group-details-panel/group-details-panel-actions';

export const groupMemberActionSet: ContextMenuActionItemSet = [
    [
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openGroupMemberAttributes(resources[0].uuid));
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
                 dispatch<any>(openRemoveGroupMemberDialog(resources[0].uuid));
            },
        },
    ],
];
