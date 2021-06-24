// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openComputeNodeRemoveDialog, openComputeNodeAttributesDialog } from 'store/compute-nodes/compute-nodes-actions';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, AttributesIcon } from "components/icon/icon";

export const computeNodeActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openComputeNodeAttributesDialog(uuid));
    }
}, {
    name: "Advanced",
    icon: AdvancedIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openAdvancedTabDialog(uuid));
    }
}, {
    name: "Remove",
    icon: RemoveIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openComputeNodeRemoveDialog(uuid));
    }
}]];
