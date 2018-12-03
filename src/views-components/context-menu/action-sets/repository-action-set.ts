// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "~/views-components/context-menu/context-menu-action-set";
import { AdvancedIcon, RemoveIcon, ShareIcon, AttributesIcon } from "~/components/icon/icon";
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { openRepositoryAttributes, openRemoveRepositoryDialog } from "~/store/repositories/repositories-actions";
import { openSharingDialog } from "~/store/sharing-dialog/sharing-dialog-actions";

export const repositoryActionSet: ContextMenuActionSet = [[{
    name: "Attributes",
    icon: AttributesIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openRepositoryAttributes(uuid));
    }
}, {
    name: "Share",
    icon: ShareIcon,
    execute: (dispatch, { uuid }) => {
        dispatch<any>(openSharingDialog(uuid));
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
        dispatch<any>(openRemoveRepositoryDialog(uuid));
    }
}]];
