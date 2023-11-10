// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, AttributesIcon } from 'components/icon/icon';
import { openSshKeyRemoveDialog, openSshKeyAttributesDialog } from 'store/auth/auth-action-ssh';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';

export const sshKeyActionSet: ContextMenuActionItemSet = [
    [
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openSshKeyAttributesDialog(resources[0].uuid));
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
                dispatch<any>(openSshKeyRemoveDialog(resources[0].uuid));
            },
        },
    ],
];
