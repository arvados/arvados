// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { RootState } from "../store";
import { DataColumns } from "components/data-table/data-table";
import { DataExplorer } from './data-explorer-reducer';
import { ListResults } from 'services/common-service/common-service';
import { createTree } from "models/tree";
import { DataTableFilters } from "components/data-table-filters/data-table-filters-tree";

export abstract class DataExplorerMiddlewareService {
    protected readonly id: string;

    protected constructor(id: string) {
        this.id = id;
    }

    public getId() {
        return this.id;
    }

    public getColumnFilters<T>(columns: DataColumns<T>, columnName: string): DataTableFilters {
        return getDataExplorerColumnFilters(columns, columnName);
    }

    abstract requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean): Promise<void>;
}

export const getDataExplorerColumnFilters = <T>(columns: DataColumns<T>, columnName: string): DataTableFilters => {
    const column = columns.find(c => c.name === columnName);
    return column ? column.filters : createTree();
};

export const dataExplorerToListParams = (dataExplorer: DataExplorer) => ({
    limit: dataExplorer.rowsPerPage,
    offset: dataExplorer.page * dataExplorer.rowsPerPage
});

export const listResultsToDataExplorerItemsMeta = <R>({ itemsAvailable, offset, limit }: ListResults<R>) => ({
    itemsAvailable,
    page: Math.floor(offset / limit),
    rowsPerPage: limit
});
