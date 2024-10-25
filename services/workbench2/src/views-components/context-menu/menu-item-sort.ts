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
    COLLECTION = "Collection",
    COLLECTION_ADMIN = "CollectionAdmin",
    COLLECTION_DIRECTORY_ITEM = "CollectionDirectoryItem",
    COLLECTION_FILE_ITEM = "CollectionFileItem",
    COLLECTION_FILES = "CollectionFiles",
    COLLECTION_FILES_MULTIPLE = "CollectionFilesMultiple",
    COLLECTION_FILES_NOT_SELECTED = "CollectionFilesNotSelected",
    FAVORITE = "Favorite",
    FILTER_GROUP = "FilterGroup",
    FILTER_GROUP_ADMIN = "FilterGroupAdmin",
    FROZEN_PROJECT = "FrozenProject",
    FROZEN_PROJECT_ADMIN = "FrozenProjectAdmin",
    GROUPS = "Group",
    GROUP_MEMBER = "GroupMember",
    KEEP_SERVICE = "KeepService",
    LINK = "Link",
    MULTI = "Multi",
    OLD_VERSION_COLLECTION = "OldVersionCollection",
    PERMISSION_EDIT = "PermissionEdit",
    PROCESS = "Process",
    PROCESS_ADMIN = "ProcessAdmin",
    PROCESS_LOGS = "ProcessLogs",
    PROCESS_RESOURCE = "ProcessResource",
    PROJECT = "Project",
    PROJECT_ADMIN = "ProjectAdmin",
    READONLY_COLLECTION = "ReadOnlyCollection",
    READONLY_COLLECTION_DIRECTORY_ITEM = "ReadOnlyCollectionDirectoryItem",
    READONLY_COLLECTION_FILE_ITEM = "ReadOnlyCollectionFileItem",
    READONLY_COLLECTION_FILES = "ReadOnlyCollectionFiles",
    READONLY_COLLECTION_FILES_MULTIPLE = "ReadOnlyCollectionFilesMultiple",
    READONLY_PROCESS_RESOURCE = "ReadOnlyProcessResource",
    READONLY_PROJECT = "ReadOnlyProject",
    READONLY_WORKFLOW = "ReadOnlyWorkflow",
    REPOSITORY = "Repository",
    RESOURCE = "Resource",
    ROOT_PROJECT = "RootProject",
    ROOT_PROJECT_ADMIN = "RootProjectAdmin",
    RUNNING_PROCESS_ADMIN = "RunningProcessAdmin",
    RUNNING_PROCESS_RESOURCE = "RunningProcessResource",
    SEARCH_RESULTS = "SearchResults",
    SSH_KEY = "SshKey",
    TRASH = "Trash",
    TRASHED_COLLECTION = "TrashedCollection",
    USER = "User",
    USER_DETAILS = "UserDetails",
    VIRTUAL_MACHINE = "VirtualMachine",
    WORKFLOW = "Workflow",
}

const processOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_UUID,
    ContextMenuActionNames.COPY_AND_RERUN_PROCESS,
    ContextMenuActionNames.CANCEL,
    ContextMenuActionNames.EDIT_PROCESS,
    ContextMenuActionNames.REMOVE,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.OUTPUTS,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.API_DETAILS,
];

const projectOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_UUID,
    ContextMenuActionNames.SHARE,
    ContextMenuActionNames.EDIT_PROJECT,
    ContextMenuActionNames.MOVE_TO_TRASH,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.NEW_PROJECT,
    ContextMenuActionNames.MOVE_TO,
    ContextMenuActionNames.FREEZE_PROJECT,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
    ContextMenuActionNames.API_DETAILS,
];

const collectionOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_UUID,
    ContextMenuActionNames.SHARE,
    ContextMenuActionNames.EDIT_COLLECTION,
    ContextMenuActionNames.MOVE_TO_TRASH,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.MAKE_A_COPY,
    ContextMenuActionNames.MOVE_TO,
    ContextMenuActionNames.ADD_TO_FAVORITES,
    ContextMenuActionNames.ADD_TO_PUBLIC_FAVORITES,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.OPEN_WITH_3RD_PARTY_CLIENT,
    ContextMenuActionNames.API_DETAILS,
];

const workflowOrder = [
    ContextMenuActionNames.VIEW_DETAILS,
    ContextMenuActionNames.OPEN_IN_NEW_TAB,
    ContextMenuActionNames.COPY_UUID,
    ContextMenuActionNames.RUN_WORKFLOW,
    ContextMenuActionNames.DELETE_WORKFLOW,
    ContextMenuActionNames.DIVIDER,
    ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
    ContextMenuActionNames.API_DETAILS,
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
    [ContextMenuKind.READONLY_PROCESS_RESOURCE]: processOrder,

    [ContextMenuKind.PROJECT]: projectOrder,
    [ContextMenuKind.PROJECT_ADMIN]: projectOrder,
    [ContextMenuKind.READONLY_PROJECT]: projectOrder,
    [ContextMenuKind.FROZEN_PROJECT]: projectOrder,
    [ContextMenuKind.FROZEN_PROJECT_ADMIN]: projectOrder,

    [ContextMenuKind.COLLECTION]: collectionOrder,
    [ContextMenuKind.COLLECTION_ADMIN]: collectionOrder,
    [ContextMenuKind.READONLY_COLLECTION]: collectionOrder,
    [ContextMenuKind.OLD_VERSION_COLLECTION]: collectionOrder,

    [ContextMenuKind.WORKFLOW]: workflowOrder,
    [ContextMenuKind.READONLY_WORKFLOW]: workflowOrder,

    [ContextMenuKind.GROUPS]: projectOrder,

    [ContextMenuKind.FILTER_GROUP]: projectOrder,
    [ContextMenuKind.FILTER_GROUP_ADMIN]: projectOrder,

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

    return Array.from(bucketMap.values()).concat(leftovers).filter((item) => item !== null).reduce((acc, val)=>{
        return acc.at(-1)?.name === "Divider" && val.name === "Divider" ? acc : acc.concat(val)
    }, []);
};
