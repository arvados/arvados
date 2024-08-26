// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from "store/data-explorer/data-explorer-middleware-service";
import { FavoritePanelColumnNames } from "views/favorite-panel/favorite-panel";
import { RootState } from "../store";
import { getUserUuid } from "common/getuser";
import { DataColumns } from "components/data-table/data-table";
import { ServiceRepository } from "services/services";
import { FilterBuilder } from "services/api/filter-builder";
import { updateFavorites } from "../favorites/favorites-actions";
import { favoritePanelActions } from "./favorite-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { loadMissingProcessesInformation } from "store/project-panel/project-panel-run-middleware-service";
import { getDataExplorerColumnFilters } from 'store/data-explorer/data-explorer-middleware-service';
import { serializeSimpleObjectTypeFilters } from '../resource-type-filters/resource-type-filters';
import { ResourceKind } from "models/resource";
import { LinkClass } from "models/link";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { ListArguments } from "services/common-service/common-service";

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

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const uuid = getUserUuid(api.getState());
        if (!dataExplorer) {
            api.dispatch(favoritesPanelDataExplorerIsNotSet());
        } else if (!uuid || !uuid.length) {
            userNotAvailable();
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                // Get items
                const responseLinks = await this.services.linkService.list(this.getLinkParams(dataExplorer, uuid));
                const uuids = responseLinks.items.map(it => it.headUuid);

                const groupItems = await this.services.groupsService.list({
                    filters: this.getResourceFilters(dataExplorer, uuids),
                });
                const collectionItems = await this.services.collectionService.list({
                    filters: this.getResourceFilters(dataExplorer, uuids),
                });
                const processItems = await this.services.containerRequestService.list({
                    filters: this.getResourceFilters(dataExplorer, uuids),
                });

                const orderedItems = [
                    ...groupItems.items,
                    ...collectionItems.items,
                    ...processItems.items
                ];

                api.dispatch(resourcesActions.SET_RESOURCES(orderedItems));
                await api.dispatch<any>(loadMissingProcessesInformation(processItems.items));
                api.dispatch(favoritePanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(responseLinks),
                    items: orderedItems.map((resource: any) => resource.uuid),
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
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
            }
        }
    }

    // Placeholder
    async requestCount() {}
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
