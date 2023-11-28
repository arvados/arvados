// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openRunProcess, deleteWorkflow } from 'store/workflow-panel/workflow-panel-actions';
import { DetailsIcon, AdvancedIcon, OpenIcon, Link, StartIcon, TrashIcon } from 'components/icon/icon';
import { copyToClipboardAction, openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from './ms-menu-actions';

const { OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW } = MultiSelectMenuActionNames;

export const msReadOnlyWorkflowActionSet: MultiSelectMenuActionSet = [
    [
        {
            name: OPEN_IN_NEW_TAB,
            icon: OpenIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openInNewTabAction(resources[0]));
            },
        },
        {
            name: COPY_TO_CLIPBOARD,
            icon: Link,

            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            name: VIEW_DETAILS,
            icon: DetailsIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch) => {
                dispatch<any>(toggleDetailsPanel());
            },
        },
        {
            name: API_DETAILS,
            icon: AdvancedIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: RUN_WORKFLOW,
            icon: StartIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(openRunProcess(resources[0].uuid, resources[0].ownerUuid, resources[0].name));
            },
        },
    ],
];

export const msWorkflowActionSet: MultiSelectMenuActionSet = [
    [
        ...msReadOnlyWorkflowActionSet[0],
        {
            name: DELETE_WORKFLOW,
            icon: TrashIcon,
            hasAlts: false,
            isForMulti: false,
            execute: (dispatch, resources) => {
                dispatch<any>(deleteWorkflow(resources[0].uuid, resources[0].ownerUuid));
            },
        },
    ],
];

export const msWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW]);
