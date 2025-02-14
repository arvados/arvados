// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, UserPanelIcon, DetailsIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import { needsUserProfileLink } from 'store/context-menu/context-menu-filters';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';

export const UserDetailsActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.VIEW_DETAILS,
            icon: DetailsIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openDetailsPanel(resources[0].uuid));
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
            name: ContextMenuActionNames.USER_ACCOUNT,
            icon: UserPanelIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(navigateToUserProfile(resources[0].uuid));
            },
            filters: [needsUserProfileLink],
        },
    ],
];
