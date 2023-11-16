// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from 'models/resource';
import { MultiSelectMenuActionSet, MultiSelectMenuActionNames } from 'views-components/multiselect-toolbar/ms-menu-action-set';
import { msCollectionActionSet } from 'views-components/multiselect-toolbar/ms-collection-action-set';
import { msProjectActionSet } from 'views-components/multiselect-toolbar/ms-project-action-set';
import { msProcessActionSet } from 'views-components/multiselect-toolbar/ms-process-action-set';

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const {
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
} = MultiSelectMenuActionNames;

//these sets govern what actions are on the ms toolbar for each resource kind
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
const processResourceMSActionsFilter = new Set([MOVE_TO, REMOVE]);
const collectionMSActionsFilter = new Set([MAKE_A_COPY, MOVE_TO, ADD_TO_TRASH]);

const { COLLECTION, PROJECT, PROCESS } = ResourceKind;

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [PROJECT]: [msProjectActionSet, projectMSActionsFilter],
    [PROCESS]: [msProcessActionSet, processResourceMSActionsFilter],
    [COLLECTION]: [msCollectionActionSet, collectionMSActionsFilter],
};
