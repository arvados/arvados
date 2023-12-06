// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openRunProcess, deleteWorkflow } from 'store/workflow-panel/workflow-panel-actions';
import { StartIcon, TrashIcon } from 'components/icon/icon';
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from './ms-menu-actions';
import { MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-actions";

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
    icon: TrashIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(deleteWorkflow(resources[0].uuid, resources[0].ownerUuid));
    },
};

export const msWorkflowActionSet: MultiSelectMenuActionSet = [[...msCommonActionSet, msRunWorkflow, msDeleteWorkflow]];

export const msReadOnlyWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW ]);
export const msWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW]);
