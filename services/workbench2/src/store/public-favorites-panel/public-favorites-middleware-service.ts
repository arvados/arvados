// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, getDataExplorerColumnFilters, listResultsToDataExplorerItemsMeta } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { resourcesActions } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { FavoritePanelColumnNames } from 'views/favorite-panel/favorite-panel';
import { publicFavoritePanelActions } from 'store/public-favorites-panel/public-favorites-action';
import { DataColumns } from 'components/data-table/data-column';
import { serializeSimpleObjectTypeFilters } from '../resource-type-filters/resource-type-filters';
import { LinkClass, LinkResource } from 'models/link';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { ListArguments, ListResults } from 'services/common-service/common-service';
import { couldNotFetchItemsAvailable } from 'store/data-explorer/data-explorer-action';

export class PublicFavoritesMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    getTypeFilters(dataExplorer: DataExplorer) {
        const columns = dataExplorer.columns as DataColumns<string, GroupContentsResource>;
        return serializeSimpleObjectTypeFilters(getDataExplorerColumnFilters(columns, FavoritePanelColumnNames.TYPE));
    }

    getLinkFilters(dataExplorer: DataExplorer, publicProjectUuid: string): string {
        return new FilterBuilder()
            .addEqual('link_class', LinkClass.STAR)
            .addEqual('owner_uuid', publicProjectUuid)
            .addIsA("head_uuid", this.getTypeFilters(dataExplorer))
            .getFilters();
    }

    getResourceFilters(dataExplorer: DataExplorer, uuids: string[]): string {
        return new FilterBuilder()
            .addIn("uuid", uuids)
            .addILike("name", dataExplorer.searchValue)
            .addIsA("uuid", this.getTypeFilters(dataExplorer))
            .getFilters();
    }

    getLinkParams(dataExplorer: DataExplorer, publicProjectUuid: string): ListArguments {
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters: this.getLinkFilters(dataExplorer, publicProjectUuid),
            count: "none",
        };
    }

    getCountParams(dataExplorer: DataExplorer, publicProjectUuid: string): ListArguments {
        return {
            filters: this.getLinkFilters(dataExplorer, publicProjectUuid),
            limit: 0,
            count: "exact",
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(favoritesPanelDataExplorerIsNotSet());
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                const uuidPrefix = api.getState().auth.config.uuidPrefix;
                const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;

                // Get items
                const responseLinks = await this.services.linkService.list(this.getLinkParams(dataExplorer, publicProjectUuid));
                const uuids = responseLinks.items.map(it => it.headUuid);

                const orderedItems = await this.services.groupsService.contents("", {
                    filters: this.getResourceFilters(dataExplorer, uuids),
                    include: ["owner_uuid", "container_uuid"],
                });

                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.items));
                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.included));
                api.dispatch(publicFavoritePanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(responseLinks),
                    items: orderedItems.items.map(resource => resource.uuid),
                }));
                api.dispatch<any>(updatePublicFavorites(uuids));
            } catch (e) {
                api.dispatch(publicFavoritePanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchPublicFavorites());
            } finally {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
            }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const uuidPrefix = api.getState().auth.config.uuidPrefix;
        const publicProjectUuid = `${uuidPrefix}-j7d0g-publicfavorites`;

        if (criteriaChanged) {
            // Get itemsAvailable
            return this.services.linkService.list(this.getCountParams(dataExplorer, publicProjectUuid))
                .then((results: ListResults<LinkResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(publicFavoritePanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
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
