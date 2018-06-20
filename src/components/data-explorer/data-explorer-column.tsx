// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { SortDirection } from "../../components/data-table/data-column";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";


export interface DataExplorerColumn<T> {
    name: string;
    selected: boolean;
    configurable?: boolean;
    sortable?: boolean;
    sortDirection?: SortDirection;
    filterable?: boolean;
    filters?: DataTableFilterItem[];
    render: (item: T) => React.ReactElement<void>;
    renderHeader?: () => React.ReactElement<void> | null;
}
