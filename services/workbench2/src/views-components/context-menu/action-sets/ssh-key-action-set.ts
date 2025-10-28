// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, AttributesIcon, CopyIcon } from 'components/icon/icon';
import { openSshKeyRemoveDialog, openSshKeyAttributesDialog } from 'store/auth/auth-action-ssh';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { copyStringToClipboardAction } from 'store/open-in-new-tab/open-in-new-tab.actions';

export const sshKeyActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.ATTRIBUTES,
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openSshKeyAttributesDialog(resources[0].uuid));
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
                dispatch<any>(openSshKeyRemoveDialog(resources[0].uuid));
            },
        },
        {
            icon: CopyIcon,
            name: ContextMenuActionNames.COPY_UUID,
            execute: (dispatch, resources) => {
                dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
            },
        },
    ],
];
