// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openRunProcess, deleteWorkflow } from 'store/workflow-panel/workflow-panel-actions';
import { StartIcon, DeleteForever, Link } from 'components/icon/icon';
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from './ms-menu-actions';
import { MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-actions";
import { copyToClipboardAction } from 'store/open-in-new-tab/open-in-new-tab.actions';

const { OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW } = MultiSelectMenuActionNames;

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
    icon: DeleteForever,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resources) => {
        for (const resource of [...resources]){
            dispatch<any>(deleteWorkflow(resource.uuid));
        }
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
