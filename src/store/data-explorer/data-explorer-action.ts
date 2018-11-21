// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { DataColumns } from "~/components/data-table/data-table";
import { DataTableFilters } from '~/components/data-table-filters/data-table-filters-tree';

export const dataExplorerActions = unionize({
    RESET_PAGINATION: ofType<{ id: string }>(),
    REQUEST_ITEMS: ofType<{ id: string }>(),
    SET_COLUMNS: ofType<{ id: string, columns: DataColumns<any> }>(),
    SET_FILTERS: ofType<{ id: string, columnName: string, filters: DataTableFilters }>(),
    SET_ITEMS: ofType<{ id: string, items: any[], page: number, rowsPerPage: number, itemsAvailable: number }>(),
    SET_PAGE: ofType<{ id: string, page: number }>(),
    SET_ROWS_PER_PAGE: ofType<{ id: string, rowsPerPage: number }>(),
    TOGGLE_COLUMN: ofType<{ id: string, columnName: string }>(),
    TOGGLE_SORT: ofType<{ id: string, columnName: string }>(),
    SET_EXPLORER_SEARCH_VALUE: ofType<{ id: string, searchValue: string }>(),
});

export type DataExplorerAction = UnionOf<typeof dataExplorerActions>;

export const bindDataExplorerActions = (id: string) => ({
    RESET_PAGINATION: () =>
        dataExplorerActions.RESET_PAGINATION({ id }),
    REQUEST_ITEMS: () =>
        dataExplorerActions.REQUEST_ITEMS({ id }),
    SET_COLUMNS: (payload: { columns: DataColumns<any> }) =>
        dataExplorerActions.SET_COLUMNS({ ...payload, id }),
    SET_FILTERS: (payload: { columnName: string, filters: DataTableFilters }) =>
        dataExplorerActions.SET_FILTERS({ ...payload, id }),
    SET_ITEMS: (payload: { items: any[], page: number, rowsPerPage: number, itemsAvailable: number }) =>
        dataExplorerActions.SET_ITEMS({ ...payload, id }),
    SET_PAGE: (payload: { page: number }) =>
        dataExplorerActions.SET_PAGE({ ...payload, id }),
    SET_ROWS_PER_PAGE: (payload: { rowsPerPage: number }) =>
        dataExplorerActions.SET_ROWS_PER_PAGE({ ...payload, id }),
    TOGGLE_COLUMN: (payload: { columnName: string }) =>
        dataExplorerActions.TOGGLE_COLUMN({ ...payload, id }),
    TOGGLE_SORT: (payload: { columnName: string }) =>
        dataExplorerActions.TOGGLE_SORT({ ...payload, id }),
    SET_EXPLORER_SEARCH_VALUE: (payload: { searchValue: string }) =>
        dataExplorerActions.SET_EXPLORER_SEARCH_VALUE({ ...payload, id }),
});
