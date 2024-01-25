// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ActiveIcon, AdminMenuIcon, AdvancedIcon, AttributesIcon, DeactivateUserIcon, LoginAsIcon } from 'components/icon/icon';
import { MultiSelectMenuAction, MultiSelectMenuActionSet, MultiSelectMenuActionNames } from './ms-menu-actions';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openActivateDialog, openDeactivateDialog, openSetupDialog } from 'store/user-profile/user-profile-actions';
import { openUserAttributes } from 'store/users/users-actions';
import { loginAs } from 'store/users/users-actions';

const { ATTRIBUTES, API_DETAILS, SETUP_USER, ACTIVATE_USER, DEACTIVATE_USER, LOGIN_AS_USER } = MultiSelectMenuActionNames;

export const USER_ATTRIBUTES_DIALOG = 'userAttributesDialog';

const msUserAttributes: MultiSelectMenuAction = {
    name: ATTRIBUTES,
    icon: AttributesIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openUserAttributes(resources[0].uuid));
    },
};

const msAdvancedAction: MultiSelectMenuAction = {
    name: API_DETAILS,
    icon: AdvancedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
    },
};

const msActivateUser: MultiSelectMenuAction = {
    name: ACTIVATE_USER,
    icon: ActiveIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openActivateDialog(resources[0].uuid));
    },
};

const msSetupUser: MultiSelectMenuAction = {
    name: SETUP_USER,
    icon: AdminMenuIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openSetupDialog(resources[0].uuid));
    },
};

const msDeactivateUser: MultiSelectMenuAction = {
    name: DEACTIVATE_USER,
    icon: DeactivateUserIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openDeactivateDialog(resources[0].uuid));
    },
};

const msLoginAsUser: MultiSelectMenuAction = {
    name: LOGIN_AS_USER,
    icon: LoginAsIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(loginAs(resources[0].uuid));
    },
};

export const msUserActionSet: MultiSelectMenuActionSet = [[msAdvancedAction, msUserAttributes, msSetupUser, msActivateUser, msDeactivateUser, msLoginAsUser]];

export const msUserCommonActionFilter = new Set([ATTRIBUTES, API_DETAILS]);
export const msUserAdminActionFilter = new Set([ATTRIBUTES, API_DETAILS, SETUP_USER, ACTIVATE_USER, DEACTIVATE_USER, LOGIN_AS_USER]);
