// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataColumn,
    resetSortDirection,
    SortDirection,
    toggleSortDirection,
    DataColumns,
} from 'components/data-table/data-column';
import {
    DataExplorerAction,
    dataExplorerActions,
    DataTableRequestState,
} from './data-explorer-action';
import {
    DataTableFetchMode,
} from 'components/data-table/data-table';
import { DataTableFilters } from 'components/data-table-filters/data-table-filters';

export interface DataExplorer {
    fetchMode: DataTableFetchMode;
    columns: DataColumns<any>;
    items: any[];
    itemsAvailable: number;
    loadingItemsAvailable: boolean;
    page: number;
    rowsPerPage: number;
    rowsPerPageOptions: number[];
    searchValue: string;
    working?: boolean;
    requestState: DataTableRequestState;
    countRequestState: DataTableRequestState;
    isNotFound: boolean;
}

export const initialDataExplorer: DataExplorer = {
    fetchMode: DataTableFetchMode.PAGINATED,
    columns: [],
    items: [],
    itemsAvailable: 0,
    loadingItemsAvailable: false,
    page: 0,
    rowsPerPage: 50,
    rowsPerPageOptions: [10, 20, 50, 100, 200, 500],
    searchValue: '',
    requestState: DataTableRequestState.IDLE,
    countRequestState: DataTableRequestState.IDLE,
    isNotFound: false,
};

export type DataExplorerState = Record<string, DataExplorer>;

export const dataExplorerReducer = (
    state: DataExplorerState = {},
    action: DataExplorerAction
) => {
    return dataExplorerActions.match(action, {
        CLEAR: ({ id }) =>
            update(state, id, (explorer) => ({
                ...explorer,
                page: 0,
                itemsAvailable: 0,
                items: [],
            })),

        RESET_PAGINATION: ({ id }) =>
            update(state, id, (explorer) => ({ ...explorer, page: 0 })),

        SET_FETCH_MODE: ({ id, fetchMode }) =>
            update(state, id, (explorer) => ({ ...explorer, fetchMode })),

        SET_COLUMNS: ({ id, columns }) => update(state, id, setColumns(columns)),

        SET_FILTERS: ({ id, columnName, filters }) =>
            update(state, id, mapColumns(setFilters(columnName, filters))),

        SET_ITEMS: ({ id, items, itemsAvailable, page, rowsPerPage }) => (
            update(state, id, (explorer) => {
                // Reject updates to pages other than current,
                //  DataExplorer middleware should retry
                // Also reject update if DE is pending, reduces flicker and appearance of race
                const updatedPage = page || 0;
                if (explorer.page === updatedPage && explorer.requestState === DataTableRequestState.PENDING) {
                    return {
                        ...explorer,
                        items,
                        itemsAvailable: itemsAvailable || explorer.itemsAvailable,
                        page: updatedPage,
                        rowsPerPage,
                    }
                } else {
                    return explorer;
                }
            })
        ),

        SET_LOADING_ITEMS_AVAILABLE: ({ id, loadingItemsAvailable }) =>
            update(state, id, (explorer) => ({
                ...explorer,
                loadingItemsAvailable,
            })),

        SET_ITEMS_AVAILABLE: ({ id, itemsAvailable }) =>
            update(state, id, (explorer) => {
                // Ignore itemsAvailable updates if another countRequest is requested
                if (explorer.countRequestState === DataTableRequestState.PENDING) {
                    return {
                        ...explorer,
                        itemsAvailable,
                        loadingItemsAvailable: false,
                    };
                } else {
                    return explorer;
                }
            }),

        RESET_ITEMS_AVAILABLE: ({ id }) =>
            update(state, id, (explorer) => ({ ...explorer, itemsAvailable: 0 })),

        APPEND_ITEMS: ({ id, items, itemsAvailable, page, rowsPerPage }) =>
            update(state, id, (explorer) => ({
                ...explorer,
                items: explorer.items.concat(items),
                itemsAvailable: explorer.itemsAvailable + (itemsAvailable || 0),
                page,
                rowsPerPage,
            })),

        SET_PAGE: ({ id, page }) =>
            update(state, id, (explorer) => ({ ...explorer, page })),

        SET_ROWS_PER_PAGE: ({ id, rowsPerPage }) =>
            update(state, id, (explorer) => ({ ...explorer, rowsPerPage })),

        SET_EXPLORER_SEARCH_VALUE: ({ id, searchValue }) =>
            update(state, id, (explorer) => ({ ...explorer, searchValue })),

        RESET_EXPLORER_SEARCH_VALUE: ({ id }) =>
            update(state, id, (explorer) => ({ ...explorer, searchValue: '' })),

        SET_REQUEST_STATE: ({ id, requestState }) =>
            update(state, id, (explorer) => ({ ...explorer, requestState })),

        SET_COUNT_REQUEST_STATE: ({ id, countRequestState }) =>
            update(state, id, (explorer) => ({ ...explorer, countRequestState })),

        TOGGLE_SORT: ({ id, columnName }) =>
            update(state, id, mapColumns(toggleSort(columnName))),

        TOGGLE_COLUMN: ({ id, columnName }) =>
            update(state, id, mapColumns(toggleColumn(columnName))),

        SET_IS_NOT_FOUND: ({ id, isNotFound }) =>
            update(state, id, (explorer) => ({ ...explorer, isNotFound })),

        default: () => state,
    });
};
export const getDataExplorer = (state: DataExplorerState, id: string) => {
    const returnValue = state[id] || initialDataExplorer;
    return returnValue;
};

export const getSortColumn = <T>(dataExplorer: DataExplorer): DataColumn<T> | undefined =>
    dataExplorer.columns.find(
        (c: DataColumn<T>) => !!c.sort && c.sort.direction !== SortDirection.NONE
    );

const update = (
    state: DataExplorerState,
    id: string,
    updateFn: (dataExplorer: DataExplorer) => DataExplorer
) => ({ ...state, [id]: updateFn(getDataExplorer(state, id)) });

const canUpdateColumns = (
    prevColumns: DataColumns<any>,
    nextColumns: DataColumns<any>
) => {
    if (prevColumns.length !== nextColumns.length) {
        return true;
    }
    for (let i = 0; i < nextColumns.length; i++) {
        const pc = prevColumns[i];
        const nc = nextColumns[i];
        if (pc.key !== nc.key || pc.name !== nc.name) {
            return true;
        }
    }
    return false;
};

const setColumns =
    (columns: DataColumns<any>) => (dataExplorer: DataExplorer) => ({
        ...dataExplorer,
        columns: canUpdateColumns(dataExplorer.columns, columns)
            ? columns
            : dataExplorer.columns,
    });

const mapColumns =
    (mapFn: (column: DataColumn<any>) => DataColumn<any>) =>
        (dataExplorer: DataExplorer) => ({
            ...dataExplorer,
            columns: dataExplorer.columns.map(mapFn),
        });

const toggleSort = (columnName: string) => (column: DataColumn<any>) =>
    column.name === columnName
        ? toggleSortDirection(column)
        : resetSortDirection(column);

const toggleColumn = (columnName: string) => (column: DataColumn<any>) =>
    column.name === columnName
        ? { ...column, selected: !column.selected }
        : column;

const setFilters =
    (columnName: string, filters: DataTableFilters) =>
        (column: DataColumn<any>) =>
            column.name === columnName ? { ...column, filters } : column;
