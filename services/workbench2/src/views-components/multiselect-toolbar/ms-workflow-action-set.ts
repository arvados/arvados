// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openRunProcess, deleteWorkflow } from 'store/workflow-panel/workflow-panel-actions';
import { StartIcon, TrashIcon, Link } from 'components/icon/icon';
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from './ms-menu-actions';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { copyToClipboardAction } from 'store/open-in-new-tab/open-in-new-tab.actions';

const { OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW } = ContextMenuActionNames;

const msRunWorkflow: MultiSelectMenuAction = {
    name: RUN_WORKFLOW,
    icon: StartIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openRunProcess(resources[0].uuid, resources[0].ownerUuid, resources[0].name));
    },
};

const msDeleteWorkflow: MultiSelectMenuAction = {
    name: DELETE_WORKFLOW,
    icon: TrashIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(deleteWorkflow(resources[0].uuid, resources[0].ownerUuid));
    },
};

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

export const msWorkflowActionSet: MultiSelectMenuActionSet = [[...msCommonActionSet, msRunWorkflow, msDeleteWorkflow, msCopyToClipboardMenuAction]];

export const msReadOnlyWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW ]);
export const msWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW]);
