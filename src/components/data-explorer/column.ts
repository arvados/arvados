// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Column<T> {
    header: string;
    selected: boolean;
    configurable?: boolean;
    key?: React.Key;
    render: (item: T) => React.ReactElement<void>;
    renderHeader?: () => React.ReactElement<void>;
}

export const isColumnConfigurable = <T>(column: Column<T>) => {
    return column.configurable === undefined || column.configurable === true;
};