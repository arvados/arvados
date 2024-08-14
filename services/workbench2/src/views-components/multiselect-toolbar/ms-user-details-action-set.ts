// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AdvancedIcon, DetailsIcon, UserPanelIcon } from 'components/icon/icon';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { MultiSelectMenuActionSet } from './ms-menu-actions';

export const UserDetailsActionSet: MultiSelectMenuActionSet= [
    [
        {
            name: ContextMenuActionNames.VIEW_DETAILS,
            icon: DetailsIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openDetailsPanel(resources[0].uuid));
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
        {
            name: ContextMenuActionNames.API_DETAILS,
            icon: AdvancedIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
    ],
];
