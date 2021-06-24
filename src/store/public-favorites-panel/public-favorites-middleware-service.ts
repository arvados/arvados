// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, getDataExplorerColumnFilters } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { resourcesActions } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { SortDirection } from 'components/data-table/data-column';
import { OrderDirection, OrderBuilder } from 'services/api/order-builder';
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { FavoritePanelColumnNames } from 'views/favorite-panel/favorite-panel';
import { publicFavoritePanelActions } from 'store/public-favorites-panel/public-favorites-action';
import { DataColumns } from 'components/data-table/data-table';
import { serializeSimpleObjectTypeFilters } from '../resource-type-filters/resource-type-filters';
import { LinkResource, LinkClass } from 'models/link';
import { GroupContentsResource, GroupContentsResourcePrefix } from 'services/groups-service/groups-service';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';

export class PublicFavoritesMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(favoritesPanelDataExplorerIsNotSet());
        } else {
            const columns = dataExplorer.columns as DataColumns<string>;
            const sortColumn = getSortColumn(dataExplorer);
            const typeFilters = serializeSimpleObjectTypeFilters(getDataExplorerColumnFilters(columns, FavoritePanelColumnNames.TYPE));


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
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const uuidPrefix = api.getState().auth.config.uuidPrefix;
                const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;
                const responseLinks = await this.services.linkService.list({
                    limit: dataExplorer.rowsPerPage,
                    offset: dataExplorer.page * dataExplorer.rowsPerPage,
                    filters: new FilterBuilder()
                        .addEqual('link_class', LinkClass.STAR)
                        .addEqual('owner_uuid', publicProjectUuid)
                        .addIsA("head_uuid", typeFilters)
                        .getFilters()
                });
                const uuids = responseLinks.items.map(it => it.headUuid);
                const groupItems: any = await this.services.groupsService.list({
                    filters: new FilterBuilder()
                        .addIn("uuid", uuids)
                        .addILike("name", dataExplorer.searchValue)
                        .addIsA("uuid", typeFilters)
                        .getFilters()
                });
                const collectionItems: any = await this.services.collectionService.list({
                    filters: new FilterBuilder()
                        .addIn("uuid", uuids)
                        .addILike("name", dataExplorer.searchValue)
                        .addIsA("uuid", typeFilters)
                        .getFilters()
                });
                const processItems: any = await this.services.containerRequestService.list({
                    filters: new FilterBuilder()
                        .addIn("uuid", uuids)
                        .addILike("name", dataExplorer.searchValue)
                        .addIsA("uuid", typeFilters)
                        .getFilters()
                });
                const response = groupItems;
                collectionItems.items.map((it: any) => {
                    response.itemsAvailable++;
                    response.items.push(it);
                });
                processItems.items.map((it: any) => {
                    response.itemsAvailable++;
                    response.items.push(it);
                });
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(resourcesActions.SET_RESOURCES(response.items));
                api.dispatch(publicFavoritePanelActions.SET_ITEMS({
                    items: response.items.map((resource: any) => resource.uuid),
                    itemsAvailable: response.itemsAvailable,
                    page: Math.floor(response.offset / response.limit),
                    rowsPerPage: response.limit
                }));
                api.dispatch<any>(updatePublicFavorites(response.items.map((item: any) => item.uuid)));
            } catch (e) {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(publicFavoritePanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchPublicFavorites());
            }
        }
    }
}

const favoritesPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Favorites panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchPublicFavorites = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch public favorites contents.',
        kind: SnackbarKind.ERROR
    });
