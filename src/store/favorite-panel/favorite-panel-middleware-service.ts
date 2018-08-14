// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { FavoritePanelFilter, FavoritePanelColumnNames } from "~/views/favorite-panel/favorite-panel";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { FavoritePanelItem, resourceToDataItem } from "~/views/favorite-panel/favorite-panel-item";
import { FavoriteOrderBuilder } from "~/services/favorite-service/favorite-order-builder";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { FilterBuilder } from "~/common/api/filter-builder";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { favoritePanelActions } from "./favorite-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";

export class FavoritePanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = api.getState().dataExplorer[this.getId()];
        const columns = dataExplorer.columns as DataColumns<FavoritePanelItem, FavoritePanelFilter>;
        const sortColumn = dataExplorer.columns.find(
            ({ sortDirection }) => sortDirection !== undefined && sortDirection !== "none"
        );
        const typeFilters = getColumnFilters(columns, FavoritePanelColumnNames.TYPE);
        const order = FavoriteOrderBuilder.create();
        if (typeFilters.length > 0) {
            this.services.favoriteService
                .list(this.services.authService.getUuid()!, {
                    limit: dataExplorer.rowsPerPage,
                    offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    order: sortColumn!.name === FavoritePanelColumnNames.NAME
                        ? sortColumn!.sortDirection === SortDirection.ASC
                            ? order.addDesc("name")
                            : order.addAsc("name")
                        : order,
                    filters: FilterBuilder
                        .create()
                        .addIsA("headUuid", typeFilters.map(filter => filter.type))
                        .addILike("name", dataExplorer.searchValue)
                })
                .then(response => {
                    api.dispatch(favoritePanelActions.SET_ITEMS({
                        items: response.items.map(resourceToDataItem),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                    api.dispatch<any>(checkPresenceInFavorites(response.items.map(item => item.uuid)));
                });
        } else {
            api.dispatch(favoritePanelActions.SET_ITEMS({
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
