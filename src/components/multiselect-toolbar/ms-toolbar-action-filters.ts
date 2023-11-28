// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { msCollectionActionSet } from 'views-components/multiselect-toolbar/ms-collection-action-set';
import { msProjectActionSet, msReadOnlyProjectActionSet, msFilterGroupActionSet, msFrozenActionSet } from 'views-components/multiselect-toolbar/ms-project-action-set';
import { msProcessActionSet } from 'views-components/multiselect-toolbar/ms-process-action-set';
import { msWorkflowActionSet } from 'views-components/multiselect-toolbar/ms-workflow-action-set';

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const {
    ADD_TO_FAVORITES,
    ADD_TO_TRASH,
    API_DETAILS,
    COPY_AND_RERUN_PROCESS,
    COPY_TO_CLIPBOARD,
    DELETE_WORKFLOW,
    EDIT_PPROJECT,
    FREEZE_PROJECT,
    MAKE_A_COPY,
    MOVE_TO,
    NEW_PROJECT,
    OPEN_IN_NEW_TAB,
    OPEN_W_3RD_PARTY_CLIENT,
    REMOVE,
    RUN_WORKFLOW,
    SHARE,
    VIEW_DETAILS,
} = MultiSelectMenuActionNames;

const allActionNames = (actionSet: MultiSelectMenuActionSet): Set<string> => new Set(actionSet[0].map((action) => action.name));

//use allActionNames or filter manually below

const processResourceMSActionsFilter = new Set([MOVE_TO, REMOVE]);
const projectMSActionsFilter = new Set([
    ADD_TO_FAVORITES,
    ADD_TO_TRASH,
    API_DETAILS,
    COPY_AND_RERUN_PROCESS,
    COPY_TO_CLIPBOARD,
    EDIT_PPROJECT,
    FREEZE_PROJECT,
    MAKE_A_COPY,
    MOVE_TO,
    NEW_PROJECT,
    OPEN_IN_NEW_TAB,
    OPEN_W_3RD_PARTY_CLIENT,
    REMOVE,
    SHARE,
    VIEW_DETAILS,
]);
const workflowMSActionFilter = new Set([OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, RUN_WORKFLOW, DELETE_WORKFLOW]);


export enum msResourceKind {
    API_CLIENT_AUTHORIZATION = "arvados#apiClientAuthorization",
    COLLECTION = "arvados#collection",
    CONTAINER = "arvados#container",
    CONTAINER_REQUEST = "arvados#containerRequest",
    GROUP = "arvados#group",
    LINK = "arvados#link",
    LOG = "arvados#log",
    PROCESS = "arvados#containerRequest",
    PROJECT = "arvados#group",
    PROJECT_FROZEN = "arvados#group_frozen",
    PROJECT_READONLY = "arvados#group_readonly",
    REPOSITORY = "arvados#repository",
    SSH_KEY = "arvados#authorizedKeys",
    KEEP_SERVICE = "arvados#keepService",
    USER = "arvados#user",
    VIRTUAL_MACHINE = "arvados#virtualMachine",
    WORKFLOW = "arvados#workflow",
    NONE = "arvados#none"
}

const { COLLECTION, PROCESS, PROJECT, PROJECT_FROZEN, PROJECT_READONLY, WORKFLOW } = msResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [COLLECTION]: [msCollectionActionSet, allActionNames(msCollectionActionSet)],
    [PROCESS]: [msProcessActionSet, processResourceMSActionsFilter],
    [PROJECT]: [msProjectActionSet, projectMSActionsFilter],
    [PROJECT_FROZEN]: [msProjectActionSet, allActionNames(msFrozenActionSet)],
    [PROJECT_READONLY]: [msProjectActionSet, allActionNames(msReadOnlyProjectActionSet)],
    [WORKFLOW]: [msWorkflowActionSet, workflowMSActionFilter]
};

