// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, ShareIcon, AttributesIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openRepositoryAttributes, openRemoveRepositoryDialog } from 'store/repositories/repositories-actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';

export const repositoryActionSet: ContextMenuActionItemSet = [
    [
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openRepositoryAttributes(resources[0].uuid));
            },
        },
        {
            name: 'Share',
            icon: ShareIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openSharingDialog(resources[0].uuid));
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
                 dispatch<any>(openRemoveRepositoryDialog(resources[0].uuid));
            },
        },
    ],
];
