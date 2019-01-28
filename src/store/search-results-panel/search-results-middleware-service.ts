// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from '~/services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta, getDataExplorerColumnFilters } from '~/store/data-explorer/data-explorer-middleware-service';
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
import { joinFilters } from '~/services/api/filter-builder';
import { DataColumns } from '~/components/data-table/data-table';
import { serializeResourceTypeFilters } from '~/store//resource-type-filters/resource-type-filters';
import { ProjectPanelColumnNames } from '~/views/project-panel/project-panel';
import * as _ from 'lodash';

export class SearchResultsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean) {
        const state = api.getState();
        const userUuid = state.auth.user!.uuid;
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const searchValue = state.searchBar.searchValue;
        const sq = parseSearchQuery(searchValue);
        const clusterId = getSearchQueryFirstProp(sq, 'cluster');
        const sessions = getSearchSessions(clusterId, state.auth.sessions);

        if (searchValue.trim() === '') {
            return;
        }

        try {
            const nameParams = getParams(dataExplorer, searchValue, sq, 'name');

            const nameLists: ListResults<GroupContentsResource>[] = await Promise.all(sessions.map(session =>
                this.services.groupsService.contents('', nameParams, session)
            ));

            const nameItems = nameLists
                .reduce((items, list) => items.concat(list.items), [] as GroupContentsResource[]);

            const nameItemsAvailable = nameLists
                .reduce((itemsAvailable, list) => itemsAvailable + list.itemsAvailable, 0);

            const descriptionParams = getParams(dataExplorer, searchValue, sq, 'description');

            const descriptionLists: ListResults<GroupContentsResource>[] = await Promise.all(sessions.map(session =>
                this.services.groupsService.contents('', descriptionParams, session)
            ));

            const descriptionItems = descriptionLists
                .reduce((items, list) => items.concat(list.items), [] as GroupContentsResource[]);

            const descriptionItemsAvailable = descriptionLists
                .reduce((itemsAvailable, list) => itemsAvailable + list.itemsAvailable, 0);

            const items = nameItems.concat(descriptionItems);

            const uniqueItems = _.uniqBy(items, 'uuid');

            const mainList: ListResults<GroupContentsResource> = {
                ...nameParams,
                kind: '',
                items: uniqueItems,
                itemsAvailable: nameItemsAvailable + descriptionItemsAvailable
            };

            api.dispatch(updateResources(mainList.items));

            api.dispatch(criteriaChanged
                ? setItems(mainList)
                : appendItems(mainList));

        } catch {
            api.dispatch(couldNotFetchSearchResults());
        }
    }
}

const typeFilters = (columns: DataColumns<string>) => serializeResourceTypeFilters(getDataExplorerColumnFilters(columns, ProjectPanelColumnNames.TYPE));

export const getParams = (dataExplorer: DataExplorer, searchValue: string, sq: ParseSearchQuery, filter: string) => ({
    ...dataExplorerToListParams(dataExplorer),
    filters: joinFilters(
        getFilters(filter, searchValue, sq),
        typeFilters(dataExplorer.columns)),
    order: getOrder(dataExplorer, filter),
    includeTrash: true
});

const getOrder = (dataExplorer: DataExplorer, orderBy: any) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<GroupContentsResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === SearchResultsPanelColumnNames.NAME ? orderBy : "modifiedAt";
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

export const appendItems = (listResults: ListResults<GroupContentsResource>) =>
    searchResultsPanelActions.APPEND_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchSearchResults = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: `Could not fetch search results for some sessions.`,
        kind: SnackbarKind.ERROR
    });
