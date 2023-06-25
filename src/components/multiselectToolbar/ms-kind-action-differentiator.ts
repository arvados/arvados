// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from 'models/resource';
import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { collectionActionSet } from 'views-components/context-menu/action-sets/collection-action-set';
import { projectActionSet } from 'views-components/context-menu/action-sets/project-action-set';
import { processResourceActionSet } from 'views-components/context-menu/action-sets/process-resource-action-set';

export function findActionByName(name: string, actionSet: ContextMenuActionSet) {
    return actionSet[0].find((action) => action.name === name);
}

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const kindToActionSet: Record<string, ContextMenuActionSet> = {
    [COLLECTION]: collectionActionSet,
    [PROJECT]: projectActionSet,
    [PROCESS]: processResourceActionSet,
};
