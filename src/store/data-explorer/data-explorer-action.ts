// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";
import { SortDirection, DataColumn } from "../../components/data-table/data-column";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";

type WithId<T> = T & { id: string };

const actions = unionize({
    SET_COLUMNS: ofType<WithId<{ columns: Array<DataColumn<any>> }>>(),
    SET_FILTERS: ofType<WithId<{columnName: string, filters: DataTableFilterItem[]}>>(),
    SET_ITEMS: ofType<WithId<{items: any[]}>>(),
    SET_PAGE: ofType<WithId<{page: number}>>(),
    SET_ROWS_PER_PAGE: ofType<WithId<{rowsPerPage: number}>>(),
    TOGGLE_COLUMN: ofType<WithId<{ columnName: string }>>(),
    TOGGLE_SORT: ofType<WithId<{ columnName: string }>>(),
    SET_SEARCH_VALUE: ofType<WithId<{searchValue: string}>>()
}, { tag: "type", value: "payload" });

export type DataExplorerAction = UnionOf<typeof actions>;

export default actions;




