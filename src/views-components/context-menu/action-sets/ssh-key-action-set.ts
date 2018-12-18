// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, AttributesIcon } from "~/components/icon/icon";
import { openSshKeyRemoveDialog, openSshKeyAttributesDialog } from '~/store/auth/auth-action-ssh';
import { openAdvancedTabDialog } from '~/store/advanced-tab/advanced-tab';

export const sshKeyActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openSshKeyAttributesDialog(uuid));
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
        dispatch<any>(openSshKeyRemoveDialog(uuid));
    }
}]];
