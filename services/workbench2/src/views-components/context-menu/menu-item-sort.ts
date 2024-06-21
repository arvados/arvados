// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuAction } from './context-menu-action-set';
import { ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { sortByProperty } from 'common/array-utils';
import { horizontalMenuDivider, verticalMenuDivider } from './actions/context-menu-divider';
import { MultiSelectMenuAction } from 'views-components/multiselect-toolbar/ms-menu-actions';

export enum ContextMenuKind {
    API_CLIENT_AUTHORIZATION = "ApiClientAuthorization",
    ROOT_PROJECT = "RootProject",
    ROOT_PROJECT_ADMIN = "RootProjectAdmin",
    PROJECT = "Project",
    FILTER_GROUP = "FilterGroup",
    READONLY_PROJECT = "ReadOnlyProject",
    FROZEN_PROJECT = "FrozenProject",
    FROZEN_PROJECT_ADMIN = "FrozenProjectAdmin",
    PROJECT_ADMIN = "ProjectAdmin",
    FILTER_GROUP_ADMIN = "FilterGroupAdmin",
    RESOURCE = "Resource",
    FAVORITE = "Favorite",
    TRASH = "Trash",
    COLLECTION_FILES = "CollectionFiles",
    COLLECTION_FILES_MULTIPLE = "CollectionFilesMultiple",
    READONLY_COLLECTION_FILES = "ReadOnlyCollectionFiles",
    READONLY_COLLECTION_FILES_MULTIPLE = "ReadOnlyCollectionFilesMultiple",
    COLLECTION_FILES_NOT_SELECTED = "CollectionFilesNotSelected",
    COLLECTION_FILE_ITEM = "CollectionFileItem",
    COLLECTION_DIRECTORY_ITEM = "CollectionDirectoryItem",
    READONLY_COLLECTION_FILE_ITEM = "ReadOnlyCollectionFileItem",
    READONLY_COLLECTION_DIRECTORY_ITEM = "ReadOnlyCollectionDirectoryItem",
    COLLECTION = "Collection",
    COLLECTION_ADMIN = "CollectionAdmin",
    READONLY_COLLECTION = "ReadOnlyCollection",
    OLD_VERSION_COLLECTION = "OldVersionCollection",
    TRASHED_COLLECTION = "TrashedCollection",
    PROCESS = "Process",
    RUNNING_PROCESS_ADMIN = "RunningProcessAdmin",
    PROCESS_ADMIN = "ProcessAdmin",
    RUNNING_PROCESS_RESOURCE = "RunningProcessResource",
    PROCESS_RESOURCE = "ProcessResource",
    READONLY_PROCESS_RESOURCE = "ReadOnlyProcessResource",
    PROCESS_LOGS = "ProcessLogs",
    REPOSITORY = "Repository",
    SSH_KEY = "SshKey",
    VIRTUAL_MACHINE = "VirtualMachine",
    KEEP_SERVICE = "KeepService",
    USER = "User",
    USER_DETAILS = "UserDetails",
    GROUPS = "Group",
    GROUP_MEMBER = "GroupMember",
    PERMISSION_EDIT = "PermissionEdit",
    LINK = "Link",
    WORKFLOW = "Workflow",
    READONLY_WORKFLOW = "ReadOnlyWorkflow",
    SEARCH_RESULTS = "SearchResults",
    MULTI = "Multi",
}



const processOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.OUTPUTS,
    ContextMenuActionNames.API_DETAILS,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.EDIT_PROCESS,
    ContextMenuActionNames.COPY_AND_RERUN_PROCESS,
    ContextMenuActionNames.CANCEL,
    ContextMenuActionNames.REMOVE,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
];

const projectOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
    ContextMenuActionNames.API_DETAILS,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.SHARE,
    ContextMenuActionNames.NEW_PROJECT,
    ContextMenuActionNames.EDIT_PROJECT,
    ContextMenuActionNames.MOVE_TO,
    ContextMenuActionNames.MOVE_TO_TRASH,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.FREEZE_PROJECT,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
];

const collectionOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
    ContextMenuActionNames.API_DETAILS,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.SHARE,
    ContextMenuActionNames.EDIT_COLLECTION,
    ContextMenuActionNames.MOVE_TO,
    ContextMenuActionNames.MAKE_A_COPY,
    ContextMenuActionNames.MOVE_TO_TRASH,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
];

const workflowOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.API_DETAILS,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.RUN_WORKFLOW,
    ContextMenuActionNames.DELETE_WORKFLOW,
]

const rootProjectOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.USER_ACCOUNT,
    ContextMenuActionNames.API_DETAILS,
];

const defaultMultiOrder = [
    ContextMenuActionNames.MOVE_TO,
    ContextMenuActionNames.MAKE_A_COPY,
    ContextMenuActionNames.MOVE_TO_TRASH,
];

const kindToOrder: Record<string, ContextMenuActionNames[]> = {
    [ContextMenuKind.MULTI]: defaultMultiOrder,

    [ContextMenuKind.PROCESS]: processOrder,
    [ContextMenuKind.PROCESS_ADMIN]: processOrder,
    [ContextMenuKind.PROCESS_RESOURCE]: processOrder,
    [ContextMenuKind.RUNNING_PROCESS_ADMIN]: processOrder,
    [ContextMenuKind.RUNNING_PROCESS_RESOURCE]: processOrder,

    [ContextMenuKind.PROJECT]: projectOrder,
    [ContextMenuKind.PROJECT_ADMIN]: projectOrder,
    [ContextMenuKind.FROZEN_PROJECT]: projectOrder,
    [ContextMenuKind.FROZEN_PROJECT_ADMIN]: projectOrder,

    [ContextMenuKind.COLLECTION]: collectionOrder,
    [ContextMenuKind.COLLECTION_ADMIN]: collectionOrder,
    [ContextMenuKind.READONLY_COLLECTION]: collectionOrder,
    [ContextMenuKind.OLD_VERSION_COLLECTION]: collectionOrder,

    [ContextMenuKind.WORKFLOW]: workflowOrder,
    [ContextMenuKind.READONLY_WORKFLOW]: workflowOrder,

    [ContextMenuKind.ROOT_PROJECT]: rootProjectOrder,
    [ContextMenuKind.ROOT_PROJECT_ADMIN]: rootProjectOrder,
};

export const menuDirection = {
    VERTICAL: 'vertical',
    HORIZONTAL: 'horizontal'
}

export const sortMenuItems = (menuKind: ContextMenuKind, menuItems: ContextMenuAction[], orthagonality: string): ContextMenuAction[] | MultiSelectMenuAction[] => {

    const preferredOrder = kindToOrder[menuKind];
    //if no specified order, sort by name
    if (!preferredOrder) return menuItems.sort(sortByProperty("name"));

    const bucketMap = new Map();
    const leftovers: ContextMenuAction[] = [];

    // if we have multiple dividers, we need each of them to have a different "name" property
    let count = 0;

    preferredOrder.forEach((name) => {
        if (name === ContextMenuActionNames.DIVIDER) {
            count++;
            bucketMap.set(`${name}-${count}`, orthagonality === menuDirection.VERTICAL ? verticalMenuDivider : horizontalMenuDivider)
        } else {
            bucketMap.set(name, null)
        }
    });
    [...menuItems].forEach((item) => {
        if (bucketMap.has(item.name)) bucketMap.set(item.name, item);
        else leftovers.push(item);
    });

    return Array.from(bucketMap.values()).concat(leftovers).filter((item) => item !== null);
};
