// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { DataTableFilters } from "../data-table-filters/data-table-filters-tree";
import { createTree } from '~/models/tree';

export interface DataColumn<T> {
    key?: React.Key;
    name: string;
    selected: boolean;
    configurable: boolean;
    sortDirection?: SortDirection;
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
    return column.sortDirection
        ? column.sortDirection === SortDirection.ASC
            ? { ...column, sortDirection: SortDirection.DESC }
            : { ...column, sortDirection: SortDirection.ASC }
        : column;
};

export const resetSortDirection = <T>(column: DataColumn<T>): DataColumn<T> => {
    return column.sortDirection ? { ...column, sortDirection: SortDirection.NONE } : column;
};

export const createDataColumn = <T>(dataColumn: Partial<DataColumn<T>>): DataColumn<T> => ({
    key: '',
    name: '',
    selected: true,
    configurable: true,
    sortDirection: SortDirection.NONE,
    filters: createTree(),
    render: () => React.createElement('span'),
    ...dataColumn,
});
