// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";
import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { collectionActionSet } from "views-components/context-menu/action-sets/collection-action-set";
import { projectActionSet } from "views-components/context-menu/action-sets/project-action-set";
import { processResourceActionSet } from "views-components/context-menu/action-sets/process-resource-action-set";

export type TMultiselectActionsFilters = Record<string, [ContextMenuActionSet, Set<string>]>;

export const contextMenuActionConsts = {
    MAKE_A_COPY: "Make a copy",
    MOVE_TO: "Move to",
    TOGGLE_TRASH_ACTION: "ToggleTrashAction",
    COPY_TO_CLIPBOARD: "Copy to clipboard",
    COPY_AND_RERUN_PROCESS: "Copy and re-run process",
    REMOVE: "Remove",
} as const;

const { MOVE_TO, TOGGLE_TRASH_ACTION, COPY_TO_CLIPBOARD, REMOVE, MAKE_A_COPY } = contextMenuActionConsts;

//these sets govern what actions are on the ms toolbar for each resource kind
const projectMSActionsFilter = new Set([COPY_TO_CLIPBOARD, MOVE_TO, TOGGLE_TRASH_ACTION]);
const processResourceMSActionsFilter = new Set([MOVE_TO, REMOVE]);
const collectionMSActionsFilter = new Set([MAKE_A_COPY, MOVE_TO, TOGGLE_TRASH_ACTION]);

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [PROJECT]: [projectActionSet, projectMSActionsFilter],
    [PROCESS]: [processResourceActionSet, processResourceMSActionsFilter],
    [COLLECTION]: [collectionActionSet, collectionMSActionsFilter],
};
