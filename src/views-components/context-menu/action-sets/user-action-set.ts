// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import {
    AdvancedIcon,
    ProjectIcon,
    AttributesIcon,
    DeactivateUserIcon,
    UserPanelIcon,
    LoginAsIcon,
    AdminMenuIcon,
    ActiveIcon,
} from "components/icon/icon";
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { loginAs, openUserAttributes, openUserProjects } from "store/users/users-actions";
import { openSetupDialog, openDeactivateDialog, openActivateDialog } from "store/user-profile/user-profile-actions";
import { navigateToUserProfile } from "store/navigation/navigation-action";
import { canActivateUser, canDeactivateUser, canSetupUser, isAdmin, needsUserProfileLink, isOtherUser } from "store/context-menu/context-menu-filters";

export const userActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openUserAttributes(uuid));
    }
}, {
    name: "Project",
    icon: ProjectIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openUserProjects(uuid));
    }
}, {
    name: "Advanced",
    icon: AdvancedIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openAdvancedTabDialog(uuid));
    }
}, {
    name: "Account Settings",
    icon: UserPanelIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(navigateToUserProfile(uuid));
    },
    filters: [needsUserProfileLink]
}],[{
    name: "Activate User",
    icon: ActiveIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openActivateDialog(uuid));
    },
    filters: [
        isAdmin,
        canActivateUser,
    ],
}, {
    name: "Setup User",
    icon: AdminMenuIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openSetupDialog(uuid));
    },
    filters: [
        isAdmin,
        canSetupUser,
    ],
}, {
    name: "Deactivate User",
    icon: DeactivateUserIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openDeactivateDialog(uuid));
    },
    filters: [
        isAdmin,
        canDeactivateUser,
    ],
}, {
    name: "Login As User",
    icon: LoginAsIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(loginAs(uuid));
    },
    filters: [
        isAdmin,
        isOtherUser,
    ],
}]];
