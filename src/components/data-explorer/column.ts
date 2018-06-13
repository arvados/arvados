// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export interface Column<T> {
    header: string;
    selected: boolean;
    render: (item: T) => React.ReactElement<void>;
}