// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, AttributesIcon } from "~/components/icon/icon";
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { openGroupMemberAttributes, openRemoveGroupMemberDialog } from '~/store/group-details-panel/group-details-panel-actions';

export const groupMemberActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openGroupMemberAttributes(uuid));
    }
}, {
    name: "Advanced",
    icon: AdvancedIcon,
    execute: (dispatch, resource) => {
        dispatch<any>(openAdvancedTabDialog(resource.uuid));
    }
}, {
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openRemoveGroupMemberDialog(uuid));
    }
}]];