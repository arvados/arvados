// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
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
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openUserAttributes(resource.uuid)));
            },
        },
        {
            name: 'Project',
            icon: ProjectIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openUserProjects(resource.uuid)));
            },
        },
        {
            name: 'API Details',
            icon: AdvancedIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openAdvancedTabDialog(resource.uuid)));
            },
        },
        {
            name: 'Account Settings',
            icon: UserPanelIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(navigateToUserProfile(resource.uuid)));
            },
            filters: [needsUserProfileLink],
        },
    ],
    [
        {
            name: 'Activate User',
            icon: ActiveIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openActivateDialog(resource.uuid)));
            },
            filters: [isAdmin, canActivateUser],
        },
        {
            name: 'Setup User',
            icon: AdminMenuIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openSetupDialog(resource.uuid)));
            },
            filters: [isAdmin, canSetupUser],
        },
        {
            name: 'Deactivate User',
            icon: DeactivateUserIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openDeactivateDialog(resource.uuid)));
            },
            filters: [isAdmin, canDeactivateUser],
        },
        {
            name: 'Login As User',
            icon: LoginAsIcon,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(loginAs(resource.uuid)));
            },
            filters: [isAdmin, isOtherUser],
        },
    ],
];
