// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, AttributesIcon, UserPanelIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openUserAttributes } from 'store/users/users-actions';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import { needsUserProfileLink } from 'store/context-menu/context-menu-filters';

export const UserDetailsActionSet: ContextMenuActionSet = [
    [
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openUserAttributes(resources[0].uuid));
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
            name: 'User Account',
            icon: UserPanelIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(navigateToUserProfile(resources[0].uuid));
            },
            filters: [needsUserProfileLink],
        },
    ],
];
