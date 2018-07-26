// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { columns, FavoritePanelFilter, FavoritePanelColumnNames } from "../../views/favorite-panel/favorite-panel";
import { getDataExplorer } from "../data-explorer/data-explorer-reducer";
import { RootState } from "../store";
import { DataColumns } from "../../components/data-table/data-table";
import { FavoritePanelItem, resourceToDataItem } from "../../views/favorite-panel/favorite-panel-item";
import { FavoriteOrderBuilder } from "../../services/favorite-service/favorite-order-builder";
import { favoriteService } from "../../services/services";
import { SortDirection } from "../../components/data-table/data-column";
import { FilterBuilder } from "../../common/api/filter-builder";
import { LinkResource } from "../../models/link";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { favoritePanelActions } from "./favorite-panel-action";

export class FavoritePanelMiddlewareService extends DataExplorerMiddlewareService {

    private static instance: FavoritePanelMiddlewareService;

    static getInstance() {
        return FavoritePanelMiddlewareService.instance
            ? FavoritePanelMiddlewareService.instance
            : new FavoritePanelMiddlewareService();
    }

    private constructor() {
        super();
    }

    get Id() {
        return "favoritePanel";
    }

    get Columns() {
        return columns;
    }

    requestItems() {
        const state = this.api.getState() as RootState;
        const dataExplorer = getDataExplorer(state.dataExplorer, this.Id);
        const columns = dataExplorer.columns as DataColumns<FavoritePanelItem, FavoritePanelFilter>;
        const sortColumn = dataExplorer.columns.find(({ sortDirection }) => Boolean(sortDirection && sortDirection !== "none"));
        const typeFilters = getColumnFilters(columns, FavoritePanelColumnNames.TYPE);
        const order = FavoriteOrderBuilder.create();
        if (typeFilters.length > 0) {
            favoriteService
                .list(state.projects.currentItemId, {
                    limit: dataExplorer.rowsPerPage,
                    offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    order: sortColumn!.name === FavoritePanelColumnNames.NAME
                        ? sortColumn!.sortDirection === SortDirection.ASC
                            ? order.addDesc("name")
                            : order.addAsc("name")
                        : order,
                    filters: FilterBuilder
                        .create<LinkResource>()
                        .addIsA("headUuid", typeFilters.map(filter => filter.type))
                        .addILike("name", dataExplorer.searchValue)
                })
                .then(response => {
                    this.api.dispatch(favoritePanelActions.SET_ITEMS({
                        items: response.items.map(resourceToDataItem),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                    this.api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
                });
        } else {
            this.api.dispatch(favoritePanelActions.SET_ITEMS({
                items: [],
                itemsAvailable: 0,
                page: 0,
                rowsPerPage: dataExplorer.rowsPerPage
            }));
        }
    }
}

const getColumnFilters = (columns: DataColumns<FavoritePanelItem, FavoritePanelFilter>, columnName: string) => {
    const column = columns.find(c => c.name === columnName);
    return column && column.filters ? column.filters.filter(f => f.selected) : [];
};
