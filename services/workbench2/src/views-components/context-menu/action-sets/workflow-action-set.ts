// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { openRunProcess, openRemoveWorkflowDialog } from "store/workflow-panel/workflow-panel-actions";
import { DetailsIcon, AdvancedIcon, OpenIcon, Link, StartIcon, DeleteForever } from "components/icon/icon";
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { toggleDetailsPanel } from "store/details-panel/details-panel-action";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";

export const readOnlyWorkflowActionSet: ContextMenuActionSet = [
    [
        {
            icon: OpenIcon,
            name: "Open in new tab",
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            icon: Link,
            name: "Copy to clipboard",
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            icon: DetailsIcon,
            name: "View details",
            execute: dispatch => {
                dispatch<any>(toggleDetailsPanel());
            },
        },
        {
            icon: AdvancedIcon,
            name: "API Details",
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            icon: StartIcon,
            name: "Run Workflow",
            execute: (dispatch, resources) => {
                dispatch<any>(openRunProcess(resources[0].uuid, resources[0].ownerUuid, resources[0].name));
            },
        },
    ],
];

export const workflowActionSet: ContextMenuActionSet = [
    [
        ...readOnlyWorkflowActionSet[0],
        {
            icon: DeleteForever,
            name: "Delete Workflow",
            execute: (dispatch, resources) => {
                dispatch<any>(openRemoveWorkflowDialog(resources[0], resources.length));
            },
        },
    ],
];
