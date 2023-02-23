// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { openRunProcess } from "store/workflow-panel/workflow-panel-actions";
import {
    RenameIcon,
    ShareIcon,
    MoveToIcon,
    CopyIcon,
    DetailsIcon,
    AdvancedIcon,
    OpenIcon,
    Link,
    RestoreVersionIcon,
    FolderSharedIcon,
    StartIcon
} from "components/icon/icon";
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";

export const workflowActionSet: ContextMenuActionSet = [[
    {
        icon: OpenIcon,
        name: "Open in new tab",
        execute: (dispatch, resource) => {
            dispatch<any>(openInNewTabAction(resource));
        }
    },
    {
        icon: Link,
        name: "Copy to clipboard",
        execute: (dispatch, resource) => {
            dispatch<any>(copyToClipboardAction(resource));
        }
    },
    {
        icon: DetailsIcon,
        name: "View details",
        execute: dispatch => {
            dispatch<any>(toggleDetailsPanel());
        }
    },
    {
        icon: AdvancedIcon,
        name: "API Details",
        execute: (dispatch, resource) => {
            dispatch<any>(openAdvancedTabDialog(resource.uuid));
        }
    },
    {
        icon: StartIcon,
        name: "Run Workflow",
        execute: (dispatch, resource) => {
            dispatch<any>(openRunProcess(resource.uuid, resource.ownerUuid, resource.name));
        }
    },
]];
