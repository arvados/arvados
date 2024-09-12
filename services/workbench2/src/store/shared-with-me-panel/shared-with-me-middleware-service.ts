// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    listResultsToDataExplorerItemsMeta,
    dataExplorerToListParams,
    getDataExplorerColumnFilters,
} from "../data-explorer/data-explorer-middleware-service";
import { ServiceRepository } from "services/services";
import { MiddlewareAPI, Dispatch } from "redux";
import { RootState } from 'store/store';
import { getDataExplorer, DataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateFavorites } from 'store/favorites/favorites-actions';
import { updateResources } from 'store/resources/resources-actions';
import { loadMissingProcessesInformation } from 'store/project-panel/project-panel-run-middleware-service';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { sharedWithMePanelActions } from './shared-with-me-panel-actions';
import { ListResults } from 'services/common-service/common-service';
import { ContentsArguments, GroupContentsResource, GroupContentsResourcePrefix } from 'services/groups-service/groups-service';
import { SortDirection } from 'components/data-table/data-column';
import { OrderBuilder, OrderDirection } from 'services/api/order-builder';
import { ProjectResource } from 'models/project';
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { AuthState } from 'store/auth/auth-reducer';
import { SharedWithMePanelColumnNames } from 'views/shared-with-me-panel/shared-with-me-panel';
import { buildProcessStatusFilters, serializeResourceTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { DataColumns } from 'components/data-table/data-table';
import { couldNotFetchItemsAvailable } from 'store/data-explorer/data-explorer-action';

export class SharedWithMeMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
            const response = await this.services.groupsService
                .contents('', getParams(dataExplorer, state.auth));
            api.dispatch<any>(updateFavorites(response.items.map(item => item.uuid)));
            api.dispatch<any>(updatePublicFavorites(response.items.map(item => item.uuid)));
            api.dispatch(updateResources(response.items));
            api.dispatch(updateResources(response.included));
            api.dispatch(setItems(response));
        } catch (e) {
            api.dispatch(couldNotFetchSharedItems());
        } finally {
            api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        if (criteriaChanged) {
            // Get itemsAvailable
            return this.services.groupsService.contents('', getCountParams(dataExplorer, state.auth))
                .then((results: ListResults<GroupContentsResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(sharedWithMePanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                })
        }
    }
}

export const getParams = (dataExplorer: DataExplorer, authState: AuthState): ContentsArguments => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer, authState),
    excludeHomeProject: true,
    count: "none",
    include: ["owner_uuid", "container_uuid"]
});

const getCountParams = (dataExplorer: DataExplorer, authState: AuthState): ContentsArguments => ({
    limit: 0,
    count: 'exact',
    filters: getFilters(dataExplorer, authState),
    excludeHomeProject: true,
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn<ProjectResource>(dataExplorer);
    const order = new OrderBuilder<ProjectResource>();
    if (sortColumn && sortColumn.sort) {
        const sortDirection = sortColumn.sort.direction === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        // Use createdAt as a secondary sort column so we break ties consistently.
        return order
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.PROCESS)
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.PROJECT)
            .addOrder(OrderDirection.DESC, "createdAt", GroupContentsResourcePrefix.PROCESS)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

const getFilters = (dataExplorer: DataExplorer, authState: AuthState) => {
    const columns = dataExplorer.columns as DataColumns<string, ProjectResource>;
    const typeFilters = serializeResourceTypeFilters(getDataExplorerColumnFilters(columns, SharedWithMePanelColumnNames.TYPE));
    const statusColumnFilters = getDataExplorerColumnFilters(columns, "Status");
    const activeStatusFilter = Object.keys(statusColumnFilters).find(filterName => statusColumnFilters[filterName].selected);

    // TODO: Extract group contents name filter
    const nameFilters = new FilterBuilder()
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
        .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
        .getFilters();

    // Filter by container status
    const statusFilters = buildProcessStatusFilters(new FilterBuilder(), activeStatusFilter || "", GroupContentsResourcePrefix.PROCESS).getFilters();

    // Filter public favorites
    const favoritesFilter = new FilterBuilder().addDistinct('uuid', `${authState.config.uuidPrefix}-j7d0g-publicfavorites`).getFilters()

    return joinFilters(statusFilters, typeFilters, nameFilters, favoritesFilter);
};


export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    sharedWithMePanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchSharedItems = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch shared items.',
        kind: SnackbarKind.ERROR
    });
