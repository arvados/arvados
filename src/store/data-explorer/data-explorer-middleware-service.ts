// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { DataTableFilterItem } from "~/components/data-table-filters/data-table-filters";
import { DataExplorer } from './data-explorer-reducer';
import { ListArguments, ListResults } from '~/common/api/common-resource-service';

export abstract class DataExplorerMiddlewareService {
    protected readonly id: string;

    protected constructor(id: string) {
        this.id = id;
    }

    public getId() {
        return this.id;
    }

    public getColumnFilters<T, F extends DataTableFilterItem>(columns: DataColumns<T, F>, columnName: string): F[] {
        const column = columns.find(c => c.name === columnName);
        return column ? column.filters.filter(f => f.selected) : [];
    }

    abstract requestItems(api: MiddlewareAPI<Dispatch, RootState>): void;
}

export const getDataExplorerColumnFilters = <T, F extends DataTableFilterItem>(columns: DataColumns<T, F>, columnName: string): F[] => {
    const column = columns.find(c => c.name === columnName);
    return column ? column.filters.filter(f => f.selected) : [];
};

export const dataExplorerToListParams = <R>(dataExplorer: DataExplorer) => ({
    limit: dataExplorer.rowsPerPage,
    offset: dataExplorer.page * dataExplorer.rowsPerPage,
});

export const listResultsToDataExplorerItemsMeta = <R>({ itemsAvailable, offset, limit }: ListResults<R>) => ({
    itemsAvailable,
    page: Math.floor(offset / limit),
    rowsPerPage: limit
});