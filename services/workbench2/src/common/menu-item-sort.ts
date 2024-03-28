// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction } from '../views-components/context-menu/context-menu-action-set';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';

const {
    ACCOUNT_SETTINGS,
    ACTIVATE_USER,
    ADD_TO_FAVORITES,
    ADD_TO_PUBLIC_FAVORITES,
    ATTRIBUTES,
    API_DETAILS,
    CANCEL,
    COPY_AND_RERUN_PROCESS,
    COPY_ITEM_INTO_EXISTING_COLLECTION,
    COPY_ITEM_INTO_NEW_COLLECTION,
    COPY_SELECTED_INTO_EXISTING_COLLECTION,
    COPY_SELECTED_INTO_SEPARATE_COLLECTIONS,
    COPY_SELECTED_INTO_NEW_COLLECTION,
    COPY_TO_CLIPBOARD,
    DEACTIVATE_USER,
    DELETE_WORKFLOW,
    DOWNLOAD,
    EDIT_COLLECTION,
    EDIT_PROCESS,
    EDIT_PROJECT,
    FREEZE_PROJECT,
    HOME_PROJECT,
    LOGIN_AS_USER,
    MAKE_A_COPY,
    MANAGE,
    MOVE_ITEM_INTO_EXISTING_COLLECTION,
    MOVE_ITEM_INTO_NEW_COLLECTION,
    MOVE_SELECTED_INTO_EXISTING_COLLECTION,
    MOVE_SELECTED_INTO_NEW_COLLECTION,
    MOVE_SELECTED_INTO_SEPARATE_COLLECTIONS,
    MOVE_TO,
    MOVE_TO_TRASH,
    NEW_COLLECTION,
    NEW_PROJECT,
    OPEN_IN_NEW_TAB,
    OPEN_WITH_3RD_PARTY_CLIENT,
    OUTPUTS,
    PROVENANCE_GRAPH,
    READ,
    REMOVE,
    REMOVE_SELECTED,
    RENAME,
    RESTORE,
    RESTORE_VERSION,
    RUN_WORKFLOW,
    SELECT_ALL,
    SETUP_USER,
    SHARE,
    UNSELECT_ALL,
    VIEW_DETAILS,
    WRITE,
} = ContextMenuActionNames;

const processOrder = [VIEW_DETAILS, OPEN_IN_NEW_TAB, OUTPUTS, API_DETAILS, EDIT_PROCESS, COPY_AND_RERUN_PROCESS, MOVE_TO, REMOVE, ADD_TO_FAVORITES, ADD_TO_PUBLIC_FAVORITES];

const kindToOrder: Record<string, ContextMenuActionNames[]> = {
    "ProcessResource": processOrder,
};

export const sortMenuItems = (menuKind:string, menuItems: ContextMenuAction[]) => {
    const order = kindToOrder[menuKind] || [];
    const bucketMap = new Map();
    order.forEach((name) => bucketMap.set(name, null));
    menuItems.forEach((item) => {if (bucketMap.has(item.name)) bucketMap.set(item.name, item)});
    console.log(Array.from(bucketMap.values()));
};