// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AdvancedIcon, AttributesIcon, UserPanelIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openUserAttributes } from 'store/users/users-actions';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { MultiSelectMenuActionSet } from './ms-menu-actions';

export const UserDetailsActionSet: MultiSelectMenuActionSet= [
    [
        {
            name: ContextMenuActionNames.ATTRIBUTES,
            icon: AttributesIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openUserAttributes(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.API_DETAILS,
            icon: AdvancedIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.USER_ACCOUNT,
            icon: UserPanelIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(navigateToUserProfile(resources[0].uuid));
            },
        },
    ],
];
