// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataTableFilterItem } from "../data-table-filters/data-table-filters";
import * as React from "react";

export interface DataColumn<T, F extends DataTableFilterItem = DataTableFilterItem> {
    name: string;
    selected: boolean;
    configurable: boolean;
    key?: React.Key;
    sortDirection?: SortDirection;
    filters?: F[];
    render: (item: T) => React.ReactElement<any>;
    renderHeader?: () => React.ReactElement<any>;
    width?: string;
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
