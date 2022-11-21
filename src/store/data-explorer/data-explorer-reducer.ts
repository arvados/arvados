// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
  DataColumn,
  resetSortDirection,
  SortDirection,
  toggleSortDirection,
} from 'components/data-table/data-column';
import {
  DataExplorerAction,
  dataExplorerActions,
  DataTableRequestState,
} from './data-explorer-action';
import {
  DataColumns,
  DataTableFetchMode,
} from 'components/data-table/data-table';
import { DataTableFilters } from 'components/data-table-filters/data-table-filters-tree';

export interface DataExplorer {
  fetchMode: DataTableFetchMode;
  columns: DataColumns<any>;
  items: any[];
  itemsAvailable: number;
  page: number;
  rowsPerPage: number;
  rowsPerPageOptions: number[];
  searchValue: string;
  working?: boolean;
  requestState: DataTableRequestState;
}

export const initialDataExplorer: DataExplorer = {
  fetchMode: DataTableFetchMode.PAGINATED,
  columns: [],
  items: [],
  itemsAvailable: 0,
  page: 0,
  rowsPerPage: 50,
  rowsPerPageOptions: [10, 20, 50, 100, 200, 500],
  searchValue: '',
  requestState: DataTableRequestState.IDLE,
};

export type DataExplorerState = Record<string, DataExplorer>;

export const dataExplorerReducer = (
  state: DataExplorerState = {},
  action: DataExplorerAction
) => {
  //   console.log('DATA_EXPLORERE_REDUCER, satate:', state);
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

    SET_ITEMS: ({ id, items, itemsAvailable, page, rowsPerPage }) =>
      update(state, id, (explorer) => ({
        ...explorer,
        items,
        itemsAvailable,
        page: page || 0,
        rowsPerPage,
      })),

    APPEND_ITEMS: ({ id, items, itemsAvailable, page, rowsPerPage }) =>
      update(state, id, (explorer) => ({
        ...explorer,
        items: state[id].items.concat(items),
        itemsAvailable: state[id].itemsAvailable + itemsAvailable,
        page,
        rowsPerPage,
      })),

    SET_PAGE: ({ id, page }) =>
      update(state, id, (explorer) => ({ ...explorer, page })),

    SET_ROWS_PER_PAGE: ({ id, rowsPerPage }) =>
      update(state, id, (explorer) => ({ ...explorer, rowsPerPage })),

    SET_EXPLORER_SEARCH_VALUE: ({ id, searchValue }) =>
      update(state, id, (explorer) => ({ ...explorer, searchValue })),

    SET_REQUEST_STATE: ({ id, requestState }) =>
      update(state, id, (explorer) => ({ ...explorer, requestState })),

    TOGGLE_SORT: ({ id, columnName }) =>
      update(state, id, mapColumns(toggleSort(columnName))),

    TOGGLE_COLUMN: ({ id, columnName }) =>
      update(state, id, mapColumns(toggleColumn(columnName))),

    default: () => state,
  });
};
export const getDataExplorer = (state: DataExplorerState, id: string) => {
  const returnValue = state[id] || initialDataExplorer;
  //lisa
  //   console.log('GETDATAEXPLORER RETURN:', state[id]);
  return returnValue;
};

export const getSortColumn = (dataExplorer: DataExplorer) =>
  dataExplorer.columns.find(
    (c: any) => !!c.sortDirection && c.sortDirection !== SortDirection.NONE
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
