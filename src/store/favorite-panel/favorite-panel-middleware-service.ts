// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "~/store/data-explorer/data-explorer-middleware-service";
import { FavoritePanelColumnNames } from "~/views/favorite-panel/favorite-panel";
import { RootState } from "../store";
import { getUserUuid } from "~/common/getuser";
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
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { progressIndicatorActions } from '~/store/progress-indicator/progress-indicator-actions.ts';
import { getDataExplorer } from "~/store/data-explorer/data-explorer-reducer";
import { loadMissingProcessesInformation } from "~/store/project-panel/project-panel-middleware-service";
import { getSortColumn } from "~/store/data-explorer/data-explorer-reducer";
import { getDataExplorerColumnFilters } from '~/store/data-explorer/data-explorer-middleware-service';
import { serializeSimpleObjectTypeFilters } from '../resource-type-filters/resource-type-filters';
import { ResourceKind } from "~/models/resource";

export class FavoritePanelMiddlewareService extends DataExplorerMiddlewareService {
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
                const responseLinks = await this.services.linkService.list({
                    filters: new FilterBuilder()
                        .addEqual("link_class", 'star')
                        .addEqual('tail_uuid', getUserUuid(api.getState()))
                        .addEqual('tail_kind', ResourceKind.USER)
                        .getFilters()
                }).then(results => results);
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
                await api.dispatch<any>(loadMissingProcessesInformation(response.items));
                api.dispatch(favoritePanelActions.SET_ITEMS({
                    items: response.items.map((resource: any) => resource.uuid),
                    itemsAvailable: response.itemsAvailable,
                    page: Math.floor(response.offset / response.limit),
                    rowsPerPage: response.limit
                }));
                api.dispatch<any>(updateFavorites(response.items.map((item: any) => item.uuid)));
            } catch (e) {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(favoritePanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchFavoritesContents());
            }
        }
    }
}

const favoritesPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Favorites panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchFavoritesContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch favorites contents.',
        kind: SnackbarKind.ERROR
    });
