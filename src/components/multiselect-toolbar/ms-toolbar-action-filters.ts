// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { msCollectionActionSet } from 'views-components/multiselect-toolbar/ms-collection-action-set';
import { msProjectActionSet, msProjectActionFilter, msReadOnlyProjectActionFilter, msFilterGroupActionFilter, msFrozenActionFilter } from 'views-components/multiselect-toolbar/ms-project-action-set';
import { msProcessActionSet } from 'views-components/multiselect-toolbar/ms-process-action-set';
import { msWorkflowActionSet, msWorkflowActionFilter, msReadOnlyWorkflowActionFilter } from 'views-components/multiselect-toolbar/ms-workflow-action-set';

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const {
    MOVE_TO,
    REMOVE,
} = MultiSelectMenuActionNames;

const allActionNames = (actionSet: MultiSelectMenuActionSet): Set<string> => new Set(actionSet[0].map((action) => action.name));

//use allActionNames or filter manually below

const processResourceMSActionsFilter = new Set([MOVE_TO, REMOVE]);

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
    PROJECT_FILTER = "arvados#group_filter",
    REPOSITORY = "arvados#repository",
    SSH_KEY = "arvados#authorizedKeys",
    KEEP_SERVICE = "arvados#keepService",
    USER = "arvados#user",
    VIRTUAL_MACHINE = "arvados#virtualMachine",
    WORKFLOW = "arvados#workflow",
    WORKFLOW_READONLY = "arvados#workflow_readonly",
    NONE = "arvados#none"
}

const { COLLECTION, PROCESS, PROJECT, PROJECT_FROZEN, PROJECT_READONLY, PROJECT_FILTER, WORKFLOW, WORKFLOW_READONLY } = msResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [COLLECTION]: [msCollectionActionSet, allActionNames(msCollectionActionSet)],
    [PROCESS]: [msProcessActionSet, processResourceMSActionsFilter],
    [PROJECT]: [msProjectActionSet, msProjectActionFilter],
    [PROJECT_FROZEN]: [msProjectActionSet, msFrozenActionFilter],
    [PROJECT_READONLY]: [msProjectActionSet, msReadOnlyProjectActionFilter],
    [PROJECT_FILTER]: [msProjectActionSet, msFilterGroupActionFilter],
    [WORKFLOW]: [msWorkflowActionSet, msWorkflowActionFilter],
    [WORKFLOW_READONLY]: [msWorkflowActionSet, msReadOnlyWorkflowActionFilter]
};

