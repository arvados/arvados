// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { collectionActionSet } from 'views-components/context-menu/action-sets/collection-action-set';
import { projectActionSet } from 'views-components/context-menu/action-sets/project-action-set';

export type TMultiselectActionsFilters = Record<string, [ContextMenuActionSet, Array<string>]>;

const collectionMSActionsFilter = ['Make a copy', 'Move to', 'ToggleTrashAction'];
const projectMSActionsFilter = ['Copy to clipboard', 'Move to', 'ToggleTrashAction'];

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    'arvados#collection': [collectionActionSet, collectionMSActionsFilter],
    'arvados#group': [projectActionSet, projectMSActionsFilter],
};
