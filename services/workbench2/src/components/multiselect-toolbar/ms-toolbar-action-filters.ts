// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuActionSet } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { msCollectionActionSet, msCommonCollectionActionFilter, msReadOnlyCollectionActionFilter, msWriteableCollectionActionFilter } from 'views-components/multiselect-toolbar/ms-collection-action-set';
import {
    msProjectActionSet,
    msCommonProjectActionFilter,
    msReadOnlyProjectActionFilter,
    msFilterGroupActionFilter,
    msAdminFilterGroupActionFilter,
    msFrozenProjectActionFilter,
    msAdminFrozenProjectActionFilter,
    msWriteableProjectActionFilter,
    msManageableProjectActionFilter,
    msManageableFrozenProjectActionFilter,
} from 'views-components/multiselect-toolbar/ms-project-action-set';
import { msProcessActionSet, msCommonProcessActionFilter, msAdminProcessActionFilter, msRunningProcessActionFilter, msReadOnlyProcessActionFilter } from 'views-components/multiselect-toolbar/ms-process-action-set';
import { msWorkflowActionSet, msWorkflowActionFilter, msReadOnlyWorkflowActionFilter } from 'views-components/multiselect-toolbar/ms-workflow-action-set';
import { UserDetailsActionSet } from 'views-components/multiselect-toolbar/ms-user-details-action-set';
import { msGroupActionSet } from 'views-components/multiselect-toolbar/ms-group-action-set';
import { ResourceKind } from 'models/resource';
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';

const {
    COLLECTION,
    COLLECTION_ADMIN,
    FILTER_GROUP,
    FILTER_GROUP_ADMIN,
    FROZEN_MANAGEABLE_PROJECT,
    FROZEN_PROJECT,
    FROZEN_PROJECT_ADMIN,
    GROUPS,
    MANAGEABLE_PROJECT,
    PROCESS_ADMIN,
    PROCESS_RESOURCE,
    PROJECT,
    PROJECT_ADMIN,
    READONLY_COLLECTION,
    READONLY_PROCESS_RESOURCE,
    READONLY_PROJECT,
    READONLY_WORKFLOW,
    ROOT_PROJECT,
    ROOT_PROJECT_ADMIN,
    RUNNING_PROCESS_ADMIN,
    RUNNING_PROCESS_RESOURCE,
    WORKFLOW,
    WRITEABLE_COLLECTION,
    WRITEABLE_PROJECT
} = ContextMenuKind;

export type TMultiselectActionsFilters = Record<string, [MultiSelectMenuActionSet, Set<string>]>;

const allActionNames = (actionSet: MultiSelectMenuActionSet): Set<string> => new Set(actionSet[0].map((action) => action.name));

export const multiselectActionsFilters: TMultiselectActionsFilters = {
    [COLLECTION]: [msCollectionActionSet, msCommonCollectionActionFilter],
    [COLLECTION_ADMIN]: [msCollectionActionSet, allActionNames(msCollectionActionSet)],
    [READONLY_COLLECTION]: [msCollectionActionSet, msReadOnlyCollectionActionFilter],
    [WRITEABLE_COLLECTION]: [msCollectionActionSet, msWriteableCollectionActionFilter],
    [ResourceKind.COLLECTION]: [msCollectionActionSet, msCommonCollectionActionFilter],

    [PROCESS_RESOURCE]: [msProcessActionSet, msCommonProcessActionFilter],
    [PROCESS_ADMIN]: [msProcessActionSet, msAdminProcessActionFilter],
    [RUNNING_PROCESS_RESOURCE]: [msProcessActionSet, msRunningProcessActionFilter],
    [RUNNING_PROCESS_ADMIN]: [msProcessActionSet, allActionNames(msProcessActionSet)],
    [READONLY_PROCESS_RESOURCE]: [msProcessActionSet, msReadOnlyProcessActionFilter],
    [ResourceKind.PROCESS]: [msProcessActionSet, msCommonProcessActionFilter],
    
    [PROJECT]: [msProjectActionSet, msCommonProjectActionFilter],
    [PROJECT_ADMIN]: [msProjectActionSet, allActionNames(msProjectActionSet)],
    [FROZEN_PROJECT]: [msProjectActionSet, msFrozenProjectActionFilter],
    [FROZEN_PROJECT_ADMIN]: [msProjectActionSet, msAdminFrozenProjectActionFilter], 
    [READONLY_PROJECT]: [msProjectActionSet, msReadOnlyProjectActionFilter],
    [WRITEABLE_PROJECT]: [msProjectActionSet, msWriteableProjectActionFilter],
    [MANAGEABLE_PROJECT]: [msProjectActionSet, msManageableProjectActionFilter],
    [FROZEN_MANAGEABLE_PROJECT]: [msProjectActionSet, msManageableFrozenProjectActionFilter],
    [ResourceKind.PROJECT]: [msProjectActionSet, msCommonProjectActionFilter],
    
    [FILTER_GROUP]: [msProjectActionSet, msFilterGroupActionFilter],
    [FILTER_GROUP_ADMIN]: [msProjectActionSet, msAdminFilterGroupActionFilter],

    [GROUPS]: [msGroupActionSet, allActionNames(msGroupActionSet)],
    
    [WORKFLOW]: [msWorkflowActionSet, msWorkflowActionFilter],
    [READONLY_WORKFLOW]: [msWorkflowActionSet, msReadOnlyWorkflowActionFilter],

    [ROOT_PROJECT]: [UserDetailsActionSet, allActionNames(UserDetailsActionSet)],
    [ROOT_PROJECT_ADMIN]: [UserDetailsActionSet, allActionNames(UserDetailsActionSet)],
    [ResourceKind.WORKFLOW]: [msWorkflowActionSet, msWorkflowActionFilter],
};
