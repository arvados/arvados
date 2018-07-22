// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";
import { DataColumns } from "../../components/data-table/data-table";

export const dataExplorerActions = unionize({
    RESET_PAGINATION: ofType<{ id: string }>(),
    REQUEST_ITEMS: ofType<{ id: string }>(),
    SET_COLUMNS: ofType<{ id: string, columns: DataColumns<any> }>(),
    SET_FILTERS: ofType<{ id: string, columnName: string, filters: DataTableFilterItem[] }>(),
    SET_ITEMS: ofType<{ id: string, items: any[], page: number, rowsPerPage: number, itemsAvailable: number }>(),
    SET_PAGE: ofType<{ id: string, page: number }>(),
    SET_ROWS_PER_PAGE: ofType<{ id: string, rowsPerPage: number }>(),
    TOGGLE_COLUMN: ofType<{ id: string, columnName: string }>(),
    TOGGLE_SORT: ofType<{ id: string, columnName: string }>(),
    SET_SEARCH_VALUE: ofType<{ id: string, searchValue: string }>(),
}, { tag: "type", value: "payload" });

export type DataExplorerAction = UnionOf<typeof dataExplorerActions>;
