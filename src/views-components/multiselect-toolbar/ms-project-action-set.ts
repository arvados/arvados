// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { MoveToIcon, Link } from "components/icon/icon";
import { openMoveProjectDialog } from "store/projects/project-move-actions";
import { ToggleTrashAction } from "views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "store/trash/trash-actions";
import { copyToClipboardAction } from "store/open-in-new-tab/open-in-new-tab.actions";

export const msCopyToClipboardMenuAction = {
    icon: Link,
    name: "Copy to clipboard",
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const msMoveToAction = {
    icon: MoveToIcon,
    name: "Move to",
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const msToggleTrashAction = {
    component: ToggleTrashAction,
    name: "ToggleTrashAction",
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!, resources.length > 1));
        }
    },
};

export const msProjectActionSet: ContextMenuActionSet = [[msCopyToClipboardMenuAction, msMoveToAction, msToggleTrashAction]];
