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
    }
},], [{
    name: "Activate User",
    adminOnly: true,
    icon: ActiveIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openActivateDialog(uuid));
    }
},{
    name: "Setup User",
    adminOnly: true,
    icon: AdminMenuIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openSetupDialog(uuid));
    }
}, {
    name: "Deactivate User",
    adminOnly: true,
    icon: DeactivateUserIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openDeactivateDialog(uuid));
    }
}, {
    name: "Login As User",
    adminOnly: true,
    icon: LoginAsIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(loginAs(uuid));
    }
},

]];
