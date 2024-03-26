// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuActionSet } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { msCollectionActionSet, msCommonCollectionActionFilter, msReadOnlyCollectionActionFilter } from 'views-components/multiselect-toolbar/ms-collection-action-set';
import {
    msProjectActionSet,
    msCommonProjectActionFilter,
    msReadOnlyProjectActionFilter,
    msFilterGroupActionFilter,
    msAdminFilterGroupActionFilter,
    msFrozenProjectActionFilter,
    msAdminFrozenProjectActionFilter
} from 'views-components/multiselect-toolbar/ms-project-action-set';
import { msProcessActionSet, msCommonProcessActionFilter, msAdminProcessActionFilter, msRunningProcessActionFilter } from 'views-components/multiselect-toolbar/ms-process-action-set';
import { msWorkflowActionSet, msWorkflowActionFilter, msReadOnlyWorkflowActionFilter } from 'views-components/multiselect-toolbar/ms-workflow-action-set';
import { ResourceKind } from 'models/resource';

export enum msMenuResourceKind {
    API_CLIENT_AUTHORIZATION = 'ApiClientAuthorization',
    ROOT_PROJECT = 'RootProject',
    PROJECT = 'Project',
    FILTER_GROUP = 'FilterGroup',
    READONLY_PROJECT = 'ReadOnlyProject',
    FROZEN_PROJECT = 'FrozenProject',
    FROZEN_PROJECT_ADMIN = 'FrozenProjectAdmin',
    PROJECT_ADMIN = 'ProjectAdmin',
    FILTER_GROUP_ADMIN = 'FilterGroupAdmin',
    RESOURCE = 'Resource',
    FAVORITE = 'Favorite',
    TRASH = 'Trash',
    COLLECTION_FILES = 'CollectionFiles',
    COLLECTION_FILES_MULTIPLE = 'CollectionFilesMultiple',
    READONLY_COLLECTION_FILES = 'ReadOnlyCollectionFiles',
    READONLY_COLLECTION_FILES_MULTIPLE = 'ReadOnlyCollectionFilesMultiple',
    COLLECTION_FILES_NOT_SELECTED = 'CollectionFilesNotSelected',
    COLLECTION_FILE_ITEM = 'CollectionFileItem',
    COLLECTION_DIRECTORY_ITEM = 'CollectionDirectoryItem',
    READONLY_COLLECTION_FILE_ITEM = 'ReadOnlyCollectionFileItem',
    READONLY_COLLECTION_DIRECTORY_ITEM = 'ReadOnlyCollectionDirectoryItem',
    COLLECTION = 'Collection',
    COLLECTION_ADMIN = 'CollectionAdmin',
    READONLY_COLLECTION = 'ReadOnlyCollection',
    OLD_VERSION_COLLECTION = 'OldVersionCollection',
    TRASHED_COLLECTION = 'TrashedCollection',
    PROCESS = 'Process',
    RUNNING_PROCESS_ADMIN = 'RunningProcessAdmin',
    PROCESS_ADMIN = 'ProcessAdmin',
    RUNNING_PROCESS_RESOURCE = 'RunningProcessResource',
    PROCESS_RESOURCE = 'ProcessResource',
    READONLY_PROCESS_RESOURCE = 'ReadOnlyProcessResource',
    PROCESS_LOGS = 'ProcessLogs',
    REPOSITORY = 'Repository',
    SSH_KEY = 'SshKey',
    VIRTUAL_MACHINE = 'VirtualMachine',
    KEEP_SERVICE = 'KeepService',
    USER = 'User',
    GROUPS = 'Group',
    GROUP_MEMBER = 'GroupMember',
    PERMISSION_EDIT = 'PermissionEdit',
    LINK = 'Link',
    WORKFLOW = 'Workflow',
    READONLY_WORKFLOW = 'ReadOnlyWorkflow',
    SEARCH_RESULTS = 'SearchResults',
}

const {
    COLLECTION,
    COLLECTION_ADMIN,
    READONLY_COLLECTION,
    PROCESS_RESOURCE,
    RUNNING_PROCESS_RESOURCE,
    RUNNING_PROCESS_ADMIN,
    PROCESS_ADMIN,
    PROJECT,
    PROJECT_ADMIN,
    FROZEN_PROJECT,
    FROZEN_PROJECT_ADMIN,
    READONLY_PROJECT,
    FILTER_GROUP,
    FILTER_GROUP_ADMIN,
    WORKFLOW,
    READONLY_WORKFLOW,
} = msMenuResourceKind;

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const allActionNames = (actionSet: MultiSelectMenuActionSet): Set<string> => new Set(actionSet[0].map((action) => action.name));

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [COLLECTION]: [msCollectionActionSet, msCommonCollectionActionFilter],
    [COLLECTION_ADMIN]: [msCollectionActionSet, allActionNames(msCollectionActionSet)],
    [READONLY_COLLECTION]: [msCollectionActionSet, msReadOnlyCollectionActionFilter],
    [ResourceKind.COLLECTION]: [msCollectionActionSet, msCommonCollectionActionFilter],

    [PROCESS_RESOURCE]: [msProcessActionSet, msCommonProcessActionFilter],
    [PROCESS_ADMIN]: [msProcessActionSet, msAdminProcessActionFilter],
    [RUNNING_PROCESS_RESOURCE]: [msProcessActionSet, msRunningProcessActionFilter],
    [RUNNING_PROCESS_ADMIN]: [msProcessActionSet, allActionNames(msProcessActionSet)],
    [ResourceKind.PROCESS]: [msProcessActionSet, msCommonProcessActionFilter],
    
    [PROJECT]: [msProjectActionSet, msCommonProjectActionFilter],
    [PROJECT_ADMIN]: [msProjectActionSet, allActionNames(msProjectActionSet)],
    [FROZEN_PROJECT]: [msProjectActionSet, msFrozenProjectActionFilter],
    [FROZEN_PROJECT_ADMIN]: [msProjectActionSet, msAdminFrozenProjectActionFilter], 
    [READONLY_PROJECT]: [msProjectActionSet, msReadOnlyProjectActionFilter],
    [ResourceKind.PROJECT]: [msProjectActionSet, msCommonProjectActionFilter],
    
    [FILTER_GROUP]: [msProjectActionSet, msFilterGroupActionFilter],
    [FILTER_GROUP_ADMIN]: [msProjectActionSet, msAdminFilterGroupActionFilter],
    
    [WORKFLOW]: [msWorkflowActionSet, msWorkflowActionFilter],
    [READONLY_WORKFLOW]: [msWorkflowActionSet, msReadOnlyWorkflowActionFilter],
};
