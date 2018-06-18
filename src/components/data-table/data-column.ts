// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface DataColumn<T> {
    name: string;
    selected: boolean;
    configurable?: boolean;
    key?: React.Key;
    render: (item: T) => React.ReactElement<void>;
    renderHeader?: () => React.ReactElement<void>;
}

export const isColumnConfigurable = <T>(column: DataColumn<T>) => {
    return column.configurable === undefined || column.configurable;
};