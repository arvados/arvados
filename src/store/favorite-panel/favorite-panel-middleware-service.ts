// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "../data-explorer/data-explorer-middleware-service";
import { FavoritePanelColumnNames, FavoritePanelFilter } from "~/views/favorite-panel/favorite-panel";
import { RootState } from "../store";
import { DataColumns } from "~/components/data-table/data-table";
import { ServiceRepository } from "~/services/services";
import { SortDirection } from "~/components/data-table/data-column";
import { FilterBuilder } from "~/services/api/filter-builder";
import { updateFavorites } from "../favorites/favorites-actions";
import { favoritePanelActions } from "./favorite-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { OrderBuilder, OrderDirection } from "~/services/api/order-builder";
import { LinkResource } from "~/models/link";
import { GroupContentsResource, GroupContentsResourcePrefix } from "~/services/groups-service/groups-service";
import { resourcesActions } from "~/store/resources/resources-actions";
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { getDataExplorer } from "~/store/data-explorer/data-explorer-reducer";

export class FavoritePanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(favoritesPanelDataExplorerIsNotSet());
         } else {

            const columns = dataExplorer.columns as DataColumns<string, FavoritePanelFilter>;
            const sortColumn = dataExplorer.columns.find(c => c.sortDirection !== SortDirection.NONE);
            const typeFilters = this.getColumnFilters(columns, FavoritePanelColumnNames.TYPE);

            const linkOrder = new OrderBuilder<LinkResource>();
            const contentOrder = new OrderBuilder<GroupContentsResource>();

            if (sortColumn && sortColumn.name === FavoritePanelColumnNames.NAME) {
                const direction = sortColumn.sortDirection === SortDirection.ASC
                    ? OrderDirection.ASC
                    : OrderDirection.DESC;

                linkOrder.addOrder(direction, "name");
                contentOrder
                    .addOrder(direction, "name", GroupContentsResourcePrefix.COLLECTION)
                    .addOrder(direction, "name", GroupContentsResourcePrefix.PROCESS)
                    .addOrder(direction, "name", GroupContentsResourcePrefix.PROJECT);
            }

            this.services.favoriteService
                .list(this.services.authService.getUuid()!, {
                    limit: dataExplorer.rowsPerPage,
                    offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    linkOrder: linkOrder.getOrder(),
                    contentOrder: contentOrder.getOrder(),
                    filters: new FilterBuilder()
                        .addIsA("headUuid", typeFilters.map(filter => filter.type))
                        .addILike("name", dataExplorer.searchValue)
                        .getFilters()
                })
                .then(response => {
                    api.dispatch(resourcesActions.SET_RESOURCES(response.items));
                    api.dispatch(favoritePanelActions.SET_ITEMS({
                        items: response.items.map(resource => resource.uuid),
                        itemsAvailable: response.itemsAvailable,
                        page: Math.floor(response.offset / response.limit),
                        rowsPerPage: response.limit
                    }));
                    api.dispatch<any>(updateFavorites(response.items.map(item => item.uuid)));
                })
                .catch(() => {
                    api.dispatch(favoritePanelActions.SET_ITEMS({
                        items: [],
                        itemsAvailable: 0,
                        page: 0,
                        rowsPerPage: dataExplorer.rowsPerPage
                    }));
                });
        }
    }
}

const favoritesPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Favorites panel is not ready.'
    });
