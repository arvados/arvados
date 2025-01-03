// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DataTableFilters } from "../data-table-filters/data-table-filters";
import { createTree } from 'models/tree';

/**
 * @template T Type of item to be displayed in the data table
 */
export interface DataColumn<T> {
    key?: React.Key;
    name: string;
    selected: boolean;
    configurable: boolean;

    /**
     * If set to true, filters on this column will be displayed in a
     * radio group and only one filter can be selected at a time.
     */
    mutuallyExclusiveFilters?: boolean;
    sort?: {direction: SortDirection, field: keyof T};
    filters: DataTableFilters;
    render: (item: T) => React.ReactElement<any>;
    renderHeader?: () => React.ReactElement<any>;
}

export enum SortDirection {
    ASC = "asc",
    DESC = "desc",
    NONE = "none"
}

export const toggleSortDirection = <T>(column: DataColumn<T>): DataColumn<T> => {
    return column.sort
        ? column.sort.direction === SortDirection.ASC
            ? { ...column, sort: {...column.sort, direction: SortDirection.DESC} }
            : { ...column, sort: {...column.sort, direction: SortDirection.ASC} }
        : column;
};

export const resetSortDirection = <T>(column: DataColumn<T>): DataColumn<T> => {
    return column.sort ? { ...column, sort: {...column.sort, direction: SortDirection.NONE} } : column;
};

export const createDataColumn = <T>(dataColumn: Partial<DataColumn<T>>): DataColumn<T> => ({
    key: '',
    name: '',
    selected: true,
    configurable: true,
    filters: createTree(),
    render: () => React.createElement('span'),
    ...dataColumn,
});

export type DataColumns<T> = Array<DataColumn<T>>;
