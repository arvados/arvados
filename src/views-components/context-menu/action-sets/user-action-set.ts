// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, ProjectIcon, AttributesIcon } from "components/icon/icon";
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openUserAttributes, openUserProjects } from "store/users/users-actions";

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
}, /*
    // Neither of the buttons on this dialog work correctly (bugs #16114 and #16124) so hide it for now.
    {
    name: "Manage",
    icon: UserPanelIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openUserManagement(uuid));
    }
} */
]];
