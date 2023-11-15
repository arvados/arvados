// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";
import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from "views-components/multiselect-toolbar/ms-menu-action-set";
import { msCollectionActionSet } from "views-components/multiselect-toolbar/ms-collection-action-set";
import { msProjectActionSet } from "views-components/multiselect-toolbar/ms-project-action-set";
import { msProcessActionSet } from "views-components/multiselect-toolbar/ms-process-action-set";

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const { MOVE_TO, ADD_TO_TRASH: TOGGLE_TRASH_ACTION, ADD_TO_FAVORITES: TOGGLE_FAVORITE_ACTION, REMOVE, MAKE_A_COPY } = MultiSelectMenuActionNames;

//these sets govern what actions are on the ms toolbar for each resource kind
const projectMSActionsFilter = new Set([MOVE_TO, TOGGLE_TRASH_ACTION, TOGGLE_FAVORITE_ACTION]);
const processResourceMSActionsFilter = new Set([MOVE_TO, REMOVE]);
const collectionMSActionsFilter = new Set([MAKE_A_COPY, MOVE_TO, TOGGLE_TRASH_ACTION]);

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [PROJECT]: [msProjectActionSet, projectMSActionsFilter],
    [PROCESS]: [msProcessActionSet, processResourceMSActionsFilter],
    [COLLECTION]: [msCollectionActionSet, collectionMSActionsFilter],
};
