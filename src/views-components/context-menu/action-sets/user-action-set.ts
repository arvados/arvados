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
} from "components/icon/icon";
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { loginAs, openUserAttributes, openUserProjects } from "store/users/users-actions";
import { openSetupDialog, openDeactivateDialog } from "store/user-profile/user-profile-actions";

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
        dispatch<any>(openAdvancedTabDialog(uuid));
    }
}, {
    name: "Setup User",
    icon: AdminMenuIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openSetupDialog(uuid));
    }
}, {
    name: "Deactivate User",
    icon: DeactivateUserIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openDeactivateDialog(uuid));
    }
}, {
    name: "Login As User",
    icon: LoginAsIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(loginAs(uuid));
    }
},

]];
