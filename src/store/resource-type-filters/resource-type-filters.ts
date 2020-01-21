// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { difference, pipe, values, includes, __ } from 'lodash/fp';
import { createTree, setNode, TreeNodeStatus, TreeNode, Tree } from '~/models/tree';
import { DataTableFilterItem, DataTableFilters } from '~/components/data-table-filters/data-table-filters-tree';
import { ResourceKind } from '~/models/resource';
import { FilterBuilder } from '~/services/api/filter-builder';
import { getSelectedNodes } from '~/models/tree';
import { CollectionType } from '~/models/collection';
import { GroupContentsResourcePrefix } from '~/services/groups-service/groups-service';

export enum ProcessStatusFilter {
    ALL = 'All',
    RUNNING = 'Running',
    FAILED = 'Failed',
    COMPLETED = 'Completed',
    CANCELLED = 'Cancelled',
    LOCKED = 'Locked',
    QUEUED = 'Queued'
}

export enum ObjectTypeFilter {
    PROJECT = 'Project',
    PROCESS = 'Process',
    COLLECTION = 'Data Collection',
}

export enum CollectionTypeFilter {
    GENERAL_COLLECTION = 'General',
    OUTPUT_COLLECTION = 'Output',
    LOG_COLLECTION = 'Log',
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
);

export const getInitialResourceTypeFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    initFilter(ObjectTypeFilter.PROJECT),
    initFilter(ObjectTypeFilter.PROCESS),
    initFilter(ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.GENERAL_COLLECTION, ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.OUTPUT_COLLECTION, ObjectTypeFilter.COLLECTION),
    initFilter(CollectionTypeFilter.LOG_COLLECTION, ObjectTypeFilter.COLLECTION),
);

export const getInitialProcessStatusFilters = pipe(
    (): DataTableFilters => createTree<DataTableFilterItem>(),
    initFilter(ProcessStatusFilter.ALL, '', true),
    initFilter(ProcessStatusFilter.RUNNING, '', false),
    initFilter(ProcessStatusFilter.FAILED, '', false),
    initFilter(ProcessStatusFilter.COMPLETED, '', false),
    initFilter(ProcessStatusFilter.CANCELLED, '', false),
    initFilter(ProcessStatusFilter.QUEUED, '', false),
    initFilter(ProcessStatusFilter.LOCKED, '', false),
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
    }
};

const serializeObjectTypeFilters = ({ fb, selectedFilters }: ReturnType<typeof createFiltersBuilder>) => {
    const collectionFilters = getMatchingFilters(values(CollectionTypeFilter), selectedFilters);
    const typeFilters = pipe(
        () => new Set(getMatchingFilters(values(ObjectTypeFilter), selectedFilters)),
        set => collectionFilters.length > 0
            ? set.add(ObjectTypeFilter.COLLECTION)
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

export const serializeResourceTypeFilters = pipe(
    createFiltersBuilder,
    serializeObjectTypeFilters,
    serializeCollectionTypeFilters,
    ({ fb }) => fb.getFilters(),
);

export const serializeSimpleObjectTypeFilters = (filters: Tree<DataTableFilterItem>) => {
    return getSelectedNodes(filters)
        .map(f => f.id)
        .map(objectTypeToResourceKind);
};
