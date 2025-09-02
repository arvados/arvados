// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { getUserUuid } from "common/getuser";
import { ServiceRepository } from "services/services";
import { FilterBuilder } from "services/api/filter-builder";
import { updateFavorites } from "../favorites/favorites-actions";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { ResourceKind } from "models/resource";
import { LinkClass } from "models/link";
import { ListArguments } from "services/common-service/common-service";
import { bindDataExplorerActions } from "../data-explorer/data-explorer-action";

export const FAVORITE_PINS_ID = "favoritePins";
export const favoritePinsActions = bindDataExplorerActions(FAVORITE_PINS_ID);

export const loadFavoritePins = () => (dispatch: Dispatch) => {
    dispatch(favoritePinsActions.RESET_EXPLORER_SEARCH_VALUE());
    dispatch(favoritePinsActions.REQUEST_ITEMS());
};

export class FavoritePinsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    // Since FavoritePins does not use a data table, we can't get these types from the data table columns like we do normally
    favoriteTypes = [ResourceKind.GROUP, ResourceKind.COLLECTION, ResourceKind.CONTAINER_REQUEST, ResourceKind.WORKFLOW];

    getLinkFilters(dataExplorer: DataExplorer, uuid: string): string {
        return new FilterBuilder()
            .addEqual("link_class", LinkClass.STAR)
            .addEqual('tail_uuid', uuid)
            .addEqual('tail_kind', ResourceKind.USER)
            .addIsA("head_uuid", this.favoriteTypes)
            .getFilters();
    }

    getResourceFilters(dataExplorer: DataExplorer, uuids: string[]): string {
        return new FilterBuilder()
            .addIn("uuid", uuids)
            .addILike("name", dataExplorer.searchValue)
            .addIsA("uuid", this.favoriteTypes)
            .getFilters();
    }

    getLinkParams(dataExplorer: DataExplorer, uuid: string): ListArguments {
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters: this.getLinkFilters(dataExplorer, uuid),
            limit: 12,
            count: "none",
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const userUuid = getUserUuid(api.getState());
        if (!userUuid || !userUuid.length) {
            userNotAvailable();
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                // Get favorite links
                const responseLinks = await this.services.linkService.list(this.getLinkParams(dataExplorer, userUuid));
                const uuids = responseLinks.items.map(it => it.headUuid);

                // Get resources from links
                const orderedItems = await this.services.groupsService.contents("", {
                    filters: this.getResourceFilters(dataExplorer, uuids),
                    include: ["owner_uuid", "container_uuid"],
                });

                api.dispatch(resourcesActions.SET_RESOURCES(responseLinks.items));
                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.items));
                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems.included));
                api.dispatch(favoritePinsActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(responseLinks),
                    items: responseLinks.items.map((resource: any) => resource.uuid),
                }));
                api.dispatch<any>(updateFavorites(uuids));
            } catch (e) {
                api.dispatch(favoritePinsActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchFavoritesContents());
            } finally {
                api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
            }
        }
    }

    // Not used
    async requestCount() {}
}

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
