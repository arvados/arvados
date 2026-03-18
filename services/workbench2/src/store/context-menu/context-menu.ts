// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";

export interface ContextMenuState {
    open: boolean;
    position: ContextMenuPosition;
    resource?: ContextMenuResource;
}

export interface ContextMenuPosition {
    x: number;
    y: number;
}

export type ContextMenuResource = {
    name: string;
    uuid: string;
    ownerUuid: string;
    description?: string | null;
    kind: ResourceKind;
    menuKind: ContextMenuKind | string;
    isTrashed?: boolean;
    isEditable?: boolean;
    outputUuid?: string;
    workflowUuid?: string;
    isAdmin?: boolean;
    isFrozen?: boolean;
    storageClassesDesired?: string[];
    properties?: { [key: string]: string | string[]; };
    isMulti?: boolean;
};

export enum ContextMenuKind {
    API_CLIENT_AUTHORIZATION = "ApiClientAuthorization",
    COLLECTION = "Collection",
    COLLECTION_ADMIN = "CollectionAdmin",
    COLLECTION_DIRECTORY_ITEM = "CollectionDirectoryItem",
    COLLECTION_FILE_ITEM = "CollectionFileItem",
    COLLECTION_FILES = "CollectionFiles",
    COLLECTION_FILES_MULTIPLE = "CollectionFilesMultiple",
    COLLECTION_FILES_NOT_SELECTED = "CollectionFilesNotSelected",
    EXTERNAL_CREDENTIAL = "ExternalCredential",
    FAVORITE = "Favorite",
    FILTER_GROUP = "FilterGroup",
    FILTER_GROUP_ADMIN = "FilterGroupAdmin",
    FROZEN_MANAGEABLE_PROJECT = "FrozenManageableProject",
    FROZEN_PROJECT = "FrozenProject",
    FROZEN_PROJECT_ADMIN = "FrozenProjectAdmin",
    GROUPS = "Group",
    BUILT_IN_GROUP = "BuiltInGroup",
    GROUP_MEMBER = "GroupMember",
    KEEP_SERVICE = "KeepService",
    LINK = "Link",
    MANAGEABLE_PROJECT = "ManageableProject",
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
    WRITEABLE_COLLECTION = "WriteableCollection",
    WRITEABLE_PROJECT = "WriteableProject"
}
