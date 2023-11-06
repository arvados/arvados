// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";
import { ContextMenuActionSet } from "views-components/context-menu/context-menu-action-set";
import { msCollectionActionSet } from "views-components/multiselect-toolbar/ms-collection-action-set";
import { msProjectActionSet } from "views-components/multiselect-toolbar/ms-project-action-set";
import { msProcessActionSet } from "views-components/multiselect-toolbar/ms-process-action-set";

export function findActionByName(name: string, actionSet: ContextMenuActionSet) {
    return actionSet[0].find(action => action.name === name);
}

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const kindToActionSet: Record<string, ContextMenuActionSet> = {
    [COLLECTION]: msCollectionActionSet,
    [PROJECT]: msProjectActionSet,
    [PROCESS]: msProcessActionSet,
};
