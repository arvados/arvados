// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta, getDataExplorerColumnFilters } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateResources } from 'store/resources/resources-actions';
import { SortDirection } from 'components/data-table/data-column';
import { OrderDirection, OrderBuilder } from 'services/api/order-builder';
import { GroupContentsResource, GroupContentsResourcePrefix } from "services/groups-service/groups-service";
import { ListResults } from 'services/common-service/common-service';
import { searchResultsPanelActions } from 'store/search-results-panel/search-results-panel-actions';
import {
    getSearchSessions,
    queryToFilters,
    getAdvancedDataFromQuery
} from 'store/search-bar/search-bar-actions';
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { DataColumns } from 'components/data-table/data-table';
import { serializeResourceTypeFilters } from 'store//resource-type-filters/resource-type-filters';
import { ProjectPanelColumnNames } from 'views/project-panel/project-panel';
import { Resource, ResourceKind } from 'models/resource';
import { ContainerRequestResource } from 'models/container-request';

export class SearchResultsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const searchValue = state.searchBar.searchValue;
        const { cluster: clusterId } = getAdvancedDataFromQuery(searchValue);
        const sessions = getSearchSessions(clusterId, state.auth.sessions);

        if (searchValue.trim() === '') {
            return;
        }

        const initial = {
            itemsAvailable: 0,
            items: [] as GroupContentsResource[],
            kind: '',
            offset: 0,
            limit: 10
        };

        if (criteriaChanged) {
            api.dispatch(setItems(initial));
        }

        sessions.forEach(session => {
            const params = getParams(dataExplorer, searchValue, session.apiRevision);
            this.services.groupsService.contents('', params, session)
                .then((response) => {
                    api.dispatch(updateResources(response.items));
                    api.dispatch(appendItems(response));
                    // Request all containers for process status to be available
                    const containerRequests = response.items.filter((item) => item.kind === ResourceKind.CONTAINER_REQUEST) as ContainerRequestResource[];
                    const containerUuids = containerRequests.map(container => container.containerUuid).filter(uuid => uuid !== null) as string[];
                    containerUuids.length && this.services.containerService
                        .list({
                            filters: new FilterBuilder()
                                .addIn('uuid', containerUuids)
                                .getFilters()
                        }, false)
                        .then((containers) => {
                            api.dispatch(updateResources(containers.items));
                        });
                }).catch(() => {
                    api.dispatch(couldNotFetchSearchResults(session.clusterId));
                });
            }
        );
    }
}

const typeFilters = (columns: DataColumns<string>) => serializeResourceTypeFilters(getDataExplorerColumnFilters(columns, ProjectPanelColumnNames.TYPE));

export const getParams = (dataExplorer: DataExplorer, query: string, apiRevision: number) => ({
    ...dataExplorerToListParams(dataExplorer),
    filters: joinFilters(
        queryToFilters(query, apiRevision),
        typeFilters(dataExplorer.columns)
    ),
    order: getOrder(dataExplorer),
    includeTrash: getAdvancedDataFromQuery(query).inTrash,
    includeOldVersions: getAdvancedDataFromQuery(query).pastVersions
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<GroupContentsResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        return order
            .addOrder(sortDirection, sortColumn.name as keyof Resource, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, sortColumn.name as keyof Resource, GroupContentsResourcePrefix.PROCESS)
            .addOrder(sortDirection, sortColumn.name as keyof Resource, GroupContentsResourcePrefix.PROJECT)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

export const setItems = (listResults: ListResults<GroupContentsResource>) =>
    searchResultsPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

export const appendItems = (listResults: ListResults<GroupContentsResource>) =>
    searchResultsPanelActions.APPEND_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchSearchResults = (cluster: string) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Could not fetch search results from ${cluster}.`,
        kind: SnackbarKind.ERROR
    });
