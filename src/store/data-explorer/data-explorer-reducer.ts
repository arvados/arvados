// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataColumn, toggleSortDirection, resetSortDirection } from "../../components/data-table/data-column";
import actions, { DataExplorerAction } from "./data-explorer-action";
import { DataTableFilterItem } from "../../components/data-table-filters/data-table-filters";

interface DataExplorer {
    columns: Array<DataColumn<any>>;
    items: any[];
    page: number;
    rowsPerPage: number;
    searchValue: string;
}

export const initialDataExplorer: DataExplorer = {
    columns: [],
    items: [],
    page: 0,
    rowsPerPage: 0,
    searchValue: ""
};

export type DataExplorerState = Record<string, DataExplorer | undefined>;

const dataExplorerReducer = (state: DataExplorerState = {}, action: DataExplorerAction) =>
    actions.match(action, {
        SET_COLUMNS: ({ id, columns }) => update(state, id, setColumns(columns)),
        SET_FILTERS: ({ id, columnName, filters }) => update(state, id, mapColumns(setFilters(columnName, filters))),
        SET_ITEMS: ({ id, items }) => update(state, id, explorer => ({ ...explorer, items })),
        SET_PAGE: ({ id, page }) => update(state, id, explorer => ({ ...explorer, page })),
        SET_ROWS_PER_PAGE: ({ id, rowsPerPage }) => update(state, id, explorer => ({ ...explorer, rowsPerPage })),
        TOGGLE_SORT: ({ id, columnName }) => update(state, id, mapColumns(toggleSort(columnName))),
        TOGGLE_COLUMN: ({ id, columnName }) => update(state, id, mapColumns(toggleColumn(columnName))),
        default: () => state
    });

export default dataExplorerReducer;

export const get = (state: DataExplorerState, id: string) => state[id] || initialDataExplorer;

const update = (state: DataExplorerState, id: string, updateFn: (dataExplorer: DataExplorer) => DataExplorer) =>
    ({ ...state, [id]: updateFn(get(state, id)) });

const setColumns = (columns: Array<DataColumn<any>>) =>
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
