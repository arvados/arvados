// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from "store/data-explorer/data-explorer-middleware-service";
import { FavoritePanelColumnNames } from "views/favorite-panel/favorite-panel";
import { RootState } from "../store";
import { getUserUuid } from "common/getuser";
import { DataColumns } from "components/data-table/data-column";
import { ServiceRepository } from "services/services";
import { FilterBuilder } from "services/api/filter-builder";
import { updateFavorites } from "../favorites/favorites-actions";
import { favoritePanelActions } from "./favorite-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorsActions } from 'store/progress-indicator/progress-indicator-actions';
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { getDataExplorerColumnFilters } from 'store/data-explorer/data-explorer-middleware-service';
import { serializeSimpleObjectTypeFilters } from '../resource-type-filters/resource-type-filters';
import { ResourceKind } from "models/resource";
import { LinkClass, LinkResource } from "models/link";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { ListArguments, ListResults } from "services/common-service/common-service";
import { couldNotFetchItemsAvailable } from "store/data-explorer/data-explorer-action";

export class FavoritePanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    getTypeFilters(dataExplorer: DataExplorer) {
        const columns = dataExplorer.columns as DataColumns<string, GroupContentsResource>;
        return serializeSimpleObjectTypeFilters(getDataExplorerColumnFilters(columns, FavoritePanelColumnNames.TYPE));
    }

    getLinkFilters(dataExplorer: DataExplorer, uuid: string): string {
        return new FilterBuilder()
            .addEqual("link_class", LinkClass.STAR)
            .addEqual('tail_uuid', uuid)
            .addEqual('tail_kind', ResourceKind.USER)
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

    getLinkParams(dataExplorer: DataExplorer, uuid: string): ListArguments {
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters: this.getLinkFilters(dataExplorer, uuid),
            count: "none",
        };
    }

    getCountParams(dataExplorer: DataExplorer, uuid: string): ListArguments {
        return {
            filters: this.getLinkFilters(dataExplorer, uuid),
            limit: 0,
            count: "exact",
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const uuid = getUserUuid(api.getState());
        if (!dataExplorer) {
            api.dispatch(favoritesPanelDataExplorerIsNotSet());
        } else if (!uuid || !uuid.length) {
            userNotAvailable();
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorsActions.START_WORKING(this.getId())); }

                // Get items
                const responseLinks = await this.services.linkService.list(this.getLinkParams(dataExplorer, uuid));
                const uuids = responseLinks.items.map(it => it.headUuid);

                const orderedItems = await this.services.groupsService.contents("", {
                    filters: this.getResourceFilters(dataExplorer, uuids),
                    include: ["owner_uuid", "container_uuid"],
                });

                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.items));
                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.included));
                api.dispatch(favoritePanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(responseLinks),
                    items: orderedItems.items.map((resource: any) => resource.uuid),
                }));
                api.dispatch<any>(updateFavorites(uuids));
            } catch (e) {
                api.dispatch(favoritePanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchFavoritesContents());
            } finally {
                api.dispatch(progressIndicatorsActions.STOP_WORKING(this.getId()));
            }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const uuid = getUserUuid(api.getState());

        if (criteriaChanged && uuid && uuid.length) {
            // Get itemsAvailable
            return this.services.linkService.list(this.getCountParams(dataExplorer, uuid))
                .then((results: ListResults<LinkResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(favoritePanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
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

const couldNotFetchFavoritesContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch favorites contents.',
        kind: SnackbarKind.ERROR
    });

const userNotAvailable = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'User favorites not available.',
        kind: SnackbarKind.ERROR
    });
