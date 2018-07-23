// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataColumn, toggleSortDirection, resetSortDirection } from "../../components/data-table/data-column";
import { dataExplorerActions, DataExplorerAction } from "./data-explorer-action";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";
import { DataColumns } from "../../components/data-table/data-table";

interface DataExplorer {
    columns: DataColumns<any>;
    items: any[];
    itemsAvailable: number;
    page: number;
    rowsPerPage: number;
    rowsPerPageOptions?: number[];
    searchValue: string;
}

export const initialDataExplorer: DataExplorer = {
    columns: [],
    items: [],
    itemsAvailable: 0,
    page: 0,
    rowsPerPage: 10,
    rowsPerPageOptions: [5, 10, 25, 50],
    searchValue: ""
};

export type DataExplorerState = Record<string, DataExplorer | undefined>;

export const dataExplorerReducer = (state: DataExplorerState = {}, action: DataExplorerAction) =>
    dataExplorerActions.match(action, {
        RESET_PAGINATION: ({ id }) =>
            update(state, id, explorer => ({ ...explorer, page: 0 })),

        SET_COLUMNS: ({ id, columns }) =>
            update(state, id, setColumns(columns)),

        SET_FILTERS: ({ id, columnName, filters }) =>
            update(state, id, mapColumns(setFilters(columnName, filters))),

        SET_ITEMS: ({ id, items, itemsAvailable, page, rowsPerPage }) =>
            update(state, id, explorer => ({ ...explorer, items, itemsAvailable, page, rowsPerPage })),

        SET_PAGE: ({ id, page }) =>
            update(state, id, explorer => ({ ...explorer, page })),

        SET_ROWS_PER_PAGE: ({ id, rowsPerPage }) =>
            update(state, id, explorer => ({ ...explorer, rowsPerPage })),

        SET_SEARCH_VALUE: ({ id, searchValue }) =>
            update(state, id, explorer => ({ ...explorer, searchValue })),

        TOGGLE_SORT: ({ id, columnName }) =>
            update(state, id, mapColumns(toggleSort(columnName))),

        TOGGLE_COLUMN: ({ id, columnName }) =>
            update(state, id, mapColumns(toggleColumn(columnName))),

        default: () => state
    });

export const getDataExplorer = (state: DataExplorerState, id: string) =>
    state[id] || initialDataExplorer;

const update = (state: DataExplorerState, id: string, updateFn: (dataExplorer: DataExplorer) => DataExplorer) =>
    ({ ...state, [id]: updateFn(getDataExplorer(state, id)) });

const setColumns = (columns: DataColumns<any>) =>
    (dataExplorer: DataExplorer) =>
        ({ ...dataExplorer, columns });

const mapColumns = (mapFn: (column: DataColumn<any>) => DataColumn<any>) =>
    (dataExplorer: DataExplorer) =>
        ({ ...dataExplorer, columns: dataExplorer.columns.map(mapFn) });

const toggleSort = (columnName: string) =>
    (column: DataColumn<any>) => column.name === columnName
        ? toggleSortDirection(column)
        : resetSortDirection(column);

const toggleColumn = (columnName: string) =>
    (column: DataColumn<any>) => column.name === columnName
        ? { ...column, selected: !column.selected }
        : column;

const setFilters = (columnName: string, filters: DataTableFilterItem[]) =>
    (column: DataColumn<any>) => column.name === columnName
        ? { ...column, filters }
        : column;
