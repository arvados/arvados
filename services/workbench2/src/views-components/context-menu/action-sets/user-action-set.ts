// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import {
    AdvancedIcon,
    ProjectIcon,
    AttributesIcon,
    DeactivateUserIcon,
    UserPanelIcon,
    LoginAsIcon,
    AdminMenuIcon,
    ActiveIcon,
} from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { loginAs, openUserAttributes, openUserProjects } from 'store/users/users-actions';
import { openSetupDialog, openDeactivateDialog, openActivateDialog } from 'store/user-profile/user-profile-actions';
import { navigateToUserProfile } from 'store/navigation/navigation-action';
import {
    canActivateUser,
    canDeactivateUser,
    canSetupUser,
    isAdmin,
    needsUserProfileLink,
    isOtherUser,
} from 'store/context-menu/context-menu-filters';

export const userActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.ATTRIBUTES,
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openUserAttributes(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.HOME_PROJECT,
            icon: ProjectIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openUserProjects(resources[0].uuid));
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
            name: ContextMenuActionNames.ACCOUNT_SETTINGS,
            icon: UserPanelIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(navigateToUserProfile(resources[0].uuid));
            },
            filters: [needsUserProfileLink],
        },
    ],
    [
        {
            name: ContextMenuActionNames.ACTIVATE_USER,
            icon: ActiveIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openActivateDialog(resources[0].uuid));
            },
            filters: [isAdmin, canActivateUser],
        },
        {
            name: ContextMenuActionNames.SETUP_USER,
            icon: AdminMenuIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openSetupDialog(resources[0].uuid));
            },
            filters: [isAdmin, canSetupUser],
        },
        {
            name: ContextMenuActionNames.LOGIN_AS_USER,
            icon: LoginAsIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(loginAs(resources[0].uuid));
            },
            filters: [isAdmin, isOtherUser],
        },
        {
            name: ContextMenuActionNames.DEACTIVATE_USER,
            icon: DeactivateUserIcon,
            execute: (dispatch, resources) => {
                dispatch<any>(openDeactivateDialog(resources[0].uuid));
            },
            filters: [isAdmin, canDeactivateUser],
        },
    ],
];
