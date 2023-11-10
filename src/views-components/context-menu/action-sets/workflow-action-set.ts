// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from "views-components/context-menu/context-menu-action-set";
import { openRunProcess, deleteWorkflow } from "store/workflow-panel/workflow-panel-actions";
import { DetailsIcon, AdvancedIcon, OpenIcon, Link, StartIcon, TrashIcon } from "components/icon/icon";
import { copyToClipboardAction, openInNewTabAction } from "store/open-in-new-tab/open-in-new-tab.actions";
import { toggleDetailsPanel } from "store/details-panel/details-panel-action";
import { openAdvancedTabDialog } from "store/advanced-tab/advanced-tab";

export const readOnlyWorkflowActionSet: ContextMenuActionItemSet = [
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

export const workflowActionSet: ContextMenuActionItemSet = [
    [
        ...readOnlyWorkflowActionSet[0],
        {
            icon: TrashIcon,
            name: "Delete Workflow",
            execute: (dispatch, resources) => {
                dispatch<any>(deleteWorkflow(resources[0].uuid, resources[0].ownerUuid));
            },
        },
    ],
];
