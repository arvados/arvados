// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DataTableFilters } from "../data-table-filters/data-table-filters-tree";
import { createTree } from 'models/tree';

/**
 *
 * @template I Type of dataexplorer item reference
 * @template R Type of resource to use to restrict values of column sort.field
 */
export interface DataColumn<I, R> {
    key?: React.Key;
    name: string;
    selected: boolean;
    configurable: boolean;

    /**
     * If set to true, filters on this column will be displayed in a
     * radio group and only one filter can be selected at a time.
     */
    mutuallyExclusiveFilters?: boolean;
    sort?: {direction: SortDirection, field: keyof R};
    filters: DataTableFilters;
    render: (item: I) => React.ReactElement<any>;
    renderHeader?: () => React.ReactElement<any>;
}

export enum SortDirection {
    ASC = "asc",
    DESC = "desc",
    NONE = "none"
}

export const toggleSortDirection = <I, R>(column: DataColumn<I, R>): DataColumn<I, R> => {
    return column.sort
        ? column.sort.direction === SortDirection.ASC
            ? { ...column, sort: {...column.sort, direction: SortDirection.DESC} }
            : { ...column, sort: {...column.sort, direction: SortDirection.ASC} }
        : column;
};

export const resetSortDirection = <I, R>(column: DataColumn<I, R>): DataColumn<I, R> => {
    return column.sort ? { ...column, sort: {...column.sort, direction: SortDirection.NONE} } : column;
};

export const createDataColumn = <I, R>(dataColumn: Partial<DataColumn<I, R>>): DataColumn<I, R> => ({
    key: '',
    name: '',
    selected: true,
    configurable: true,
    filters: createTree(),
    render: () => React.createElement('span'),
    ...dataColumn,
});
