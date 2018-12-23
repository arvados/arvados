// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from '~/services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from '~/store/data-explorer/data-explorer-middleware-service';
import { RootState } from '~/store/store';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from '~/store/data-explorer/data-explorer-reducer';
import { updateResources } from '~/store/resources/resources-actions';
import { SortDirection } from '~/components/data-table/data-column';
import { SearchResultsPanelColumnNames } from '~/views/search-results-panel/search-results-panel-view';
import { OrderDirection, OrderBuilder } from '~/services/api/order-builder';
import { GroupContentsResource, GroupContentsResourcePrefix } from "~/services/groups-service/groups-service";
import { ListResults } from '~/services/common-service/common-service';
import { searchResultsPanelActions } from '~/store/search-results-panel/search-results-panel-actions';
import {
    getFilters,
    getSearchQueryFirstProp,
    getSearchSessions, ParseSearchQuery,
    parseSearchQuery
} from '~/store/search-bar/search-bar-actions';
import { getSortColumn } from "~/store/data-explorer/data-explorer-reducer";
import { Session } from "~/models/session";

export class SearchResultsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const userUuid = state.auth.user!.uuid;
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const searchValue = state.searchBar.searchValue;
        const sq = parseSearchQuery(searchValue);
        const clusterId = getSearchQueryFirstProp(sq, 'cluster');
        const sessions = getSearchSessions(clusterId, state.auth.sessions);
        sessions.forEach(async session => {
            try {
                const response = await this.services.groupsService.contents(userUuid, getParams(dataExplorer, searchValue, sq), session);
                api.dispatch(updateResources(response.items));
                api.dispatch(setItems(response));
            } catch {
                api.dispatch(couldNotFetchSearchResults(session));
            }
        });
    }
}

export const getParams = (dataExplorer: DataExplorer, searchValue: string, sq: ParseSearchQuery) => ({
    ...dataExplorerToListParams(dataExplorer),
    filters: getFilters('name', searchValue, sq),
    order: getOrder(dataExplorer)
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<GroupContentsResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === SearchResultsPanelColumnNames.NAME ? "name" : "modifiedAt";
        return order
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROCESS)
            .addOrder(sortDirection, columnName, GroupContentsResourcePrefix.PROJECT)
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

const couldNotFetchSearchResults = (session: Session) =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Could not fetch search results for ${session.clusterId}.`,
        kind: SnackbarKind.ERROR
    });
