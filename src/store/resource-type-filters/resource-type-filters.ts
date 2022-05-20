// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { difference, pipe, values, includes, __ } from 'lodash/fp';
import { createTree, setNode, TreeNodeStatus, TreeNode, Tree } from 'models/tree';
import { DataTableFilterItem, DataTableFilters } from 'components/data-table-filters/data-table-filters-tree';
import { ResourceKind } from 'models/resource';
import { FilterBuilder } from 'services/api/filter-builder';
import { getSelectedNodes } from 'models/tree';
import { CollectionType } from 'models/collection';
import { GroupContentsResourcePrefix } from 'services/groups-service/groups-service';
import { ContainerState } from 'models/container';
import { ContainerRequestState } from 'models/container-request';

export enum ProcessStatusFilter {
    ALL = 'All',
    RUNNING = 'Running',
    FAILED = 'Failed',
    COMPLETED = 'Completed',
    CANCELLED = 'Cancelled',
    ONHOLD = 'On hold',
    QUEUED = 'Queued'
}

export enum ObjectTypeFilter {
    PROJECT = 'Project',
    PROCESS = 'Process',
    COLLECTION = 'Data collection',
    WORKFLOW = 'Workflow',
}

export enum GroupTypeFilter {
    PROJECT = 'Project (normal)',
    FILTER_GROUP = 'Filter group',
}

export enum CollectionTypeFilter {
    GENERAL_COLLECTION = 'General',
    OUTPUT_COLLECTION = 'Output',
    LOG_COLLECTION = 'Log',
}

export enum ProcessTypeFilter {
    MAIN_PROCESS = 'Main',
    CHILD_PROCESS = 'Child',
}

const initFilter = (name: string, parent = '', isSelected?: boolean) =>
    setNode<DataTableFilterItem>({
        id: name,
        value: { name },
        parent,
        children: [],
        active: false,
        selected: isSelected !== undefined ? isSelected : true,
        expanded: false,
        status: TreeNodeStatus.LOADED,
    });

export const getSimpleObjectTypeFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    initFilter(ObjectTypeFilter.PROJECT),
    initFilter(ObjectTypeFilter.PROCESS),
    initFilter(ObjectTypeFilter.COLLECTION),
    initFilter(ObjectTypeFilter.WORKFLOW),
);

// Using pipe() with more than 7 arguments makes the return type be 'any',
// causing compile issues.
export const getInitialResourceTypeFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    pipe(
        initFilter(ObjectTypeFilter.PROJECT),
        initFilter(GroupTypeFilter.PROJECT, ObjectTypeFilter.PROJECT),
        initFilter(GroupTypeFilter.FILTER_GROUP, ObjectTypeFilter.PROJECT),
    ),
    pipe(
        initFilter(ObjectTypeFilter.PROCESS),
        initFilter(ProcessTypeFilter.MAIN_PROCESS, ObjectTypeFilter.PROCESS),
        initFilter(ProcessTypeFilter.CHILD_PROCESS, ObjectTypeFilter.PROCESS)
    ),
    pipe(
        initFilter(ObjectTypeFilter.COLLECTION),
        initFilter(CollectionTypeFilter.GENERAL_COLLECTION, ObjectTypeFilter.COLLECTION),
        initFilter(CollectionTypeFilter.OUTPUT_COLLECTION, ObjectTypeFilter.COLLECTION),
        initFilter(CollectionTypeFilter.LOG_COLLECTION, ObjectTypeFilter.COLLECTION),
    ),
    initFilter(ObjectTypeFilter.WORKFLOW)

);

export const getInitialProcessTypeFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    initFilter(ProcessTypeFilter.MAIN_PROCESS),
    initFilter(ProcessTypeFilter.CHILD_PROCESS, '', false)
);

export const getInitialProcessStatusFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    pipe(
        initFilter(ProcessStatusFilter.ALL, '', true),
        initFilter(ProcessStatusFilter.ONHOLD, '', false),
        initFilter(ProcessStatusFilter.QUEUED, '', false),
        initFilter(ProcessStatusFilter.RUNNING, '', false),
        initFilter(ProcessStatusFilter.COMPLETED, '', false),
        initFilter(ProcessStatusFilter.CANCELLED, '', false),
        initFilter(ProcessStatusFilter.FAILED, '', false),
    ),
);

export const getTrashPanelTypeFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    initFilter(ObjectTypeFilter.PROJECT),
    initFilter(ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.GENERAL_COLLECTION, ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.OUTPUT_COLLECTION, ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.LOG_COLLECTION, ObjectTypeFilter.COLLECTION),
);

const createFiltersBuilder = (filters: DataTableFilters) =>
    ({ fb: new FilterBuilder(), selectedFilters: getSelectedNodes(filters) });

const getMatchingFilters = (values: string[], filters: TreeNode<DataTableFilterItem>[]) =>
    filters
        .map(f => f.id)
        .filter(includes(__, values));

const objectTypeToResourceKind = (type: ObjectTypeFilter) => {
    switch (type) {
        case ObjectTypeFilter.PROJECT:
            return ResourceKind.PROJECT;
        case ObjectTypeFilter.PROCESS:
            return ResourceKind.PROCESS;
        case ObjectTypeFilter.COLLECTION:
            return ResourceKind.COLLECTION;
        case ObjectTypeFilter.WORKFLOW:
            return ResourceKind.WORKFLOW;
    }
};

const serializeObjectTypeFilters = ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => {
    const groupFilters = getMatchingFilters(values(GroupTypeFilter), selectedFilters);
    const collectionFilters = getMatchingFilters(values(CollectionTypeFilter), selectedFilters);
    const processFilters = getMatchingFilters(values(ProcessTypeFilter), selectedFilters);
    const typeFilters = pipe(
        () => new Set(getMatchingFilters(values(ObjectTypeFilter), selectedFilters)),
        set => groupFilters.length > 0
            ? set.add(ObjectTypeFilter.PROJECT)
            : set,
        set => collectionFilters.length > 0
            ? set.add(ObjectTypeFilter.COLLECTION)
            : set,
        set => processFilters.length > 0
            ? set.add(ObjectTypeFilter.PROCESS)
            : set,
        set => Array.from(set)
    )();

    return {
        fb: typeFilters.length > 0
            ? fb.addIsA('uuid', typeFilters.map(objectTypeToResourceKind))
            : fb,
        selectedFilters,
    };
};

const collectionTypeToPropertyValue = (type: CollectionTypeFilter) => {
    switch (type) {
        case CollectionTypeFilter.GENERAL_COLLECTION:
            return CollectionType.GENERAL;
        case CollectionTypeFilter.OUTPUT_COLLECTION:
            return CollectionType.OUTPUT;
        case CollectionTypeFilter.LOG_COLLECTION:
            return CollectionType.LOG;
    }
};

const serializeCollectionTypeFilters = ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => pipe(
    () => getMatchingFilters(values(CollectionTypeFilter), selectedFilters),
    filters => filters.map(collectionTypeToPropertyValue),
    mappedFilters => ({
        fb: buildCollectionTypeFilters({ fb, filters: mappedFilters }),
        selectedFilters
    })
)();

const COLLECTION_TYPES = values(CollectionType);

const NON_GENERAL_COLLECTION_TYPES = difference(COLLECTION_TYPES, [CollectionType.GENERAL]);

const COLLECTION_PROPERTIES_PREFIX = `${GroupContentsResourcePrefix.COLLECTION}.properties`;

const buildCollectionTypeFilters = ({ fb, filters }: { fb: FilterBuilder, filters: CollectionType[] }) => {
    switch (true) {
        case filters.length === 0 || filters.length === COLLECTION_TYPES.length:
            return fb;
        case includes(CollectionType.GENERAL, filters):
            return fb.addNotIn('type', difference(NON_GENERAL_COLLECTION_TYPES, filters), COLLECTION_PROPERTIES_PREFIX);
        default:
            return fb.addIn('type', filters, COLLECTION_PROPERTIES_PREFIX);
    }
};

const serializeGroupTypeFilters = ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => pipe(
    () => getMatchingFilters(values(GroupTypeFilter), selectedFilters),
    filters => filters,
    mappedFilters => ({
        fb: buildGroupTypeFilters({ fb, filters: mappedFilters, use_prefix: true }),
        selectedFilters
    })
)();

const GROUP_TYPES = values(GroupTypeFilter);

const buildGroupTypeFilters = ({ fb, filters, use_prefix }: { fb: FilterBuilder, filters: string[], use_prefix: boolean }) => {
    switch (true) {
        case filters.length === 0 || filters.length === GROUP_TYPES.length:
            return fb;
        case includes(GroupTypeFilter.PROJECT, filters):
            return fb.addEqual('groups.group_class', 'project');
        case includes(GroupTypeFilter.FILTER_GROUP, filters):
            return fb.addEqual('groups.group_class', 'filter');
        default:
            return fb;
    }
};

const serializeProcessTypeFilters = ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => pipe(
    () => getMatchingFilters(values(ProcessTypeFilter), selectedFilters),
    filters => filters,
    mappedFilters => ({
        fb: buildProcessTypeFilters({ fb, filters: mappedFilters, use_prefix: true }),
        selectedFilters
    })
)();

const PROCESS_TYPES = values(ProcessTypeFilter);
const PROCESS_PREFIX = GroupContentsResourcePrefix.PROCESS;

const buildProcessTypeFilters = ({ fb, filters, use_prefix }: { fb: FilterBuilder, filters: string[], use_prefix: boolean }) => {
    switch (true) {
        case filters.length === 0 || filters.length === PROCESS_TYPES.length:
            return fb;
        case includes(ProcessTypeFilter.MAIN_PROCESS, filters):
            return fb.addEqual('requesting_container_uuid', null, use_prefix ? PROCESS_PREFIX : '');
        case includes(ProcessTypeFilter.CHILD_PROCESS, filters):
            return fb.addDistinct('requesting_container_uuid', null, use_prefix ? PROCESS_PREFIX : '');
        default:
            return fb;
    }
};

export const serializeResourceTypeFilters = pipe(
    createFiltersBuilder,
    serializeObjectTypeFilters,
    serializeGroupTypeFilters,
    serializeCollectionTypeFilters,
    serializeProcessTypeFilters,
    ({ fb }) => fb.getFilters(),
);

export const serializeOnlyProcessTypeFilters = pipe(
    createFiltersBuilder,
    ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => pipe(
        () => getMatchingFilters(values(ProcessTypeFilter), selectedFilters),
        filters => filters,
        mappedFilters => ({
            fb: buildProcessTypeFilters({ fb, filters: mappedFilters, use_prefix: false }),
            selectedFilters
        })
    )(),
    ({ fb }) => fb.getFilters(),
);

export const serializeSimpleObjectTypeFilters = (filters: Tree<DataTableFilterItem>) => {
    return getSelectedNodes(filters)
        .map(f => f.id)
        .map(objectTypeToResourceKind);
};

export const buildProcessStatusFilters = (fb: FilterBuilder, activeStatusFilter: string, resourcePrefix?: string): FilterBuilder => {
    switch (activeStatusFilter) {
        case ProcessStatusFilter.ONHOLD: {
            fb.addDistinct('state', ContainerRequestState.FINAL, resourcePrefix);
            fb.addEqual('priority', '0', resourcePrefix);
            fb.addIn('container.state', [ContainerState.QUEUED, ContainerState.LOCKED], resourcePrefix);
            break;
        }
        case ProcessStatusFilter.COMPLETED: {
            fb.addEqual('container.state', ContainerState.COMPLETE, resourcePrefix);
            fb.addEqual('container.exit_code', '0', resourcePrefix);
            break;
        }
        case ProcessStatusFilter.FAILED: {
            fb.addEqual('container.state', ContainerState.COMPLETE, resourcePrefix);
            fb.addDistinct('container.exit_code', '0', resourcePrefix);
            break;
        }
        case ProcessStatusFilter.QUEUED: {
            fb.addEqual('container.state', ContainerState.QUEUED, resourcePrefix);
            fb.addDistinct('priority', '0', resourcePrefix);
            break;
        }
        case ProcessStatusFilter.CANCELLED:
        case ProcessStatusFilter.RUNNING: {
            fb.addEqual('container.state', activeStatusFilter, resourcePrefix);
            break;
        }
    }
    return fb;
};
