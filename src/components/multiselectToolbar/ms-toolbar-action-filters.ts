// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from 'models/resource';
import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { collectionActionSet } from 'views-components/context-menu/action-sets/collection-action-set';
import { projectActionSet } from 'views-components/context-menu/action-sets/project-action-set';
import { processResourceActionSet } from 'views-components/context-menu/action-sets/process-resource-action-set';

export type TMultiselectActionsFilters = Record<string, [ContextMenuActionSet, Set<string>]>;

export const contextMenuActionConsts = {
    MAKE_A_COPY: 'Make a copy',
    MOVE_TO: 'Move to',
    TOGGLE_TRASH_ACTION: 'ToggleTrashAction',
    COPY_TO_CLIPBOARD: 'Copy to clipboard',
    COPY_AND_RERUN_PROCESS: 'Copy and re-run process',
    REMOVE: 'Remove',
} as const;

const { MAKE_A_COPY, MOVE_TO, TOGGLE_TRASH_ACTION, COPY_TO_CLIPBOARD, COPY_AND_RERUN_PROCESS, REMOVE } =
    contextMenuActionConsts;

const collectionMSActionsFilter = new Set([MAKE_A_COPY, MOVE_TO, TOGGLE_TRASH_ACTION]);
const projectMSActionsFilter = new Set([COPY_TO_CLIPBOARD, MOVE_TO, TOGGLE_TRASH_ACTION]);
const processResourceMSActionsFilter = new Set([COPY_AND_RERUN_PROCESS, MOVE_TO, REMOVE]);

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [COLLECTION]: [collectionActionSet, collectionMSActionsFilter],
    [PROJECT]: [projectActionSet, projectMSActionsFilter],
    [PROCESS]: [processResourceActionSet, processResourceMSActionsFilter],
};
