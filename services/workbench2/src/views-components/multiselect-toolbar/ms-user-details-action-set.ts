// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { AdvancedIcon, AttributesIcon, UserPanelIcon } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { MultiSelectMenuActionSet } from './ms-menu-actions';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { dialogActions } from 'store/dialog/dialog-actions';
import { UserResource } from 'models/user';
import { getResource } from 'store/resources/resources';

const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';

const msOpenUserAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<UserResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: USER_ATTRIBUTES_DIALOG, data }));
    };

export const UserDetailsActionSet: MultiSelectMenuActionSet= [
    [
        {
            name: ContextMenuActionNames.ATTRIBUTES,
            icon: AttributesIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(msOpenUserAttributes(resources[0].uuid));
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
