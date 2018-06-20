// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface DataColumn<T> {
    name: string;
    selected: boolean;
    configurable?: boolean;
    key?: React.Key;
    sortDirection?: SortDirection;
    onSortToggle?: () => void;
    render: (item: T) => React.ReactElement<void>;
    renderHeader?: () => React.ReactElement<void> | null;
}

export type SortDirection = "asc" | "desc";

export const isColumnConfigurable = <T>(column: DataColumn<T>) => {
    return column.configurable === undefined || column.configurable;
};

export const toggleSortDirection = <T>(column: DataColumn<T>): DataColumn<T> => {
    const sortDirection = column.sortDirection === undefined || column.sortDirection === "desc" ? "asc" : "desc";
    return { ...column, sortDirection };
};

export const resetSortDirection = <T>(column: DataColumn<T>): DataColumn<T> => {
    return { ...column, sortDirection: undefined };
};
