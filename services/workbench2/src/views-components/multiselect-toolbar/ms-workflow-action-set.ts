// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openRunProcess, openRemoveWorkflowDialog } from 'store/workflow-panel/workflow-panel-actions';
import { StartIcon, DeleteForever, Link, CopyIcon } from 'components/icon/icon';
import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from './ms-menu-actions';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { copyToClipboardAction, copyStringToClipboardAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
import { ShareIcon } from 'components/icon/icon';

const { OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW, SHARE } = ContextMenuActionNames;

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
            dispatch<any>(openRemoveWorkflowDialog(resource, resources.length));
        }
    },
};

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_LINK_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

const msCopyUuid: MultiSelectMenuAction  = {
    name: COPY_UUID,
    icon: CopyIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyStringToClipboardAction(resources[0].uuid));
    },
};

const msShareAction: MultiSelectMenuAction  = {
    name: SHARE,
    icon: ShareIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openSharingDialog(resources[0].uuid));
    },
};

export const msWorkflowActionSet: MultiSelectMenuActionSet = [[...msCommonActionSet, msRunWorkflow, msDeleteWorkflow, msCopyToClipboardMenuAction, msShareAction, msCopyUuid]];

export const msReadOnlyWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW ]);
export const msWorkflowActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_LINK_TO_CLIPBOARD, COPY_UUID, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW]);
