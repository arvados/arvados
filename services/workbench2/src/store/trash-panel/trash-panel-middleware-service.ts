// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService, dataExplorerToListParams,
    listResultsToDataExplorerItemsMeta
} from "../data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { getUserUuid } from "common/getuser";
import { DataColumns } from "components/data-table/data-table";
import { ServiceRepository } from "services/services";
import { SortDirection } from "components/data-table/data-column";
import { FilterBuilder } from "services/api/filter-builder";
import { trashPanelActions } from "./trash-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { OrderBuilder, OrderDirection } from "services/api/order-builder";
import { GroupContentsResource, GroupContentsResourcePrefix } from "services/groups-service/groups-service";
import { TrashPanelColumnNames } from "views/trash-panel/trash-panel";
import { updateFavorites } from "store/favorites/favorites-actions";
import { updatePublicFavorites } from 'store/public-favorites/public-favorites-actions';
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { updateResources } from "store/resources/resources-actions";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { DataExplorer, getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { serializeResourceTypeFilters } from 'store//resource-type-filters/resource-type-filters';
import { getDataExplorerColumnFilters } from 'store/data-explorer/data-explorer-middleware-service';
import { joinFilters } from 'services/api/filter-builder';
import { CollectionResource } from "models/collection";
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { removeDisabledButton } from "store/multiselect/multiselect-actions";
export class TrashPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = api.getState().dataExplorer[this.getId()];
        const columns = dataExplorer.columns as DataColumns<string, CollectionResource>;

        const typeFilters = serializeResourceTypeFilters(getDataExplorerColumnFilters(columns, TrashPanelColumnNames.TYPE));

        const otherFilters = new FilterBuilder()
            .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.COLLECTION)
            // .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROCESS)
            .addILike("name", dataExplorer.searchValue, GroupContentsResourcePrefix.PROJECT)
            .addEqual("is_trashed", true)
            .getFilters();

        const filters = joinFilters(
            typeFilters,
            otherFilters,
        );

        const userUuid = getUserUuid(api.getState());
        if (!userUuid) { return; }
        try {
            api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
            const listResults = await this.services.groupsService
                .contents('', {
                    ...dataExplorerToListParams(dataExplorer),
                    order: getOrder(dataExplorer),
                    filters,
                    recursive: true,
                    includeTrash: true
                });
            api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));

            const items = listResults.items.map(it => it.uuid);

            api.dispatch(trashPanelActions.SET_ITEMS({
                ...listResultsToDataExplorerItemsMeta(listResults),
                items
            }));
            api.dispatch<any>(updateFavorites(items));
            api.dispatch<any>(updatePublicFavorites(items));
            api.dispatch(updateResources(listResults.items));
        } catch (e) {
            api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
            api.dispatch(trashPanelActions.SET_ITEMS({
                items: [],
                itemsAvailable: 0,
                page: 0,
                rowsPerPage: dataExplorer.rowsPerPage
            }));
            api.dispatch(couldNotFetchTrashContents());
        }
        api.dispatch<any>(removeDisabledButton(ContextMenuActionNames.MOVE_TO_TRASH))
    }
}

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn<GroupContentsResource>(dataExplorer);
    const order = new OrderBuilder<GroupContentsResource>();
    if (sortColumn && sortColumn.sort) {
        const sortDirection = sortColumn.sort.direction === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        // Use createdAt as a secondary sort column so we break ties consistently.
        return order
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.COLLECTION)
            .addOrder(sortDirection, sortColumn.sort.field, GroupContentsResourcePrefix.PROJECT)
            .addOrder(OrderDirection.DESC, "createdAt", GroupContentsResourcePrefix.PROCESS)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

const couldNotFetchTrashContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch trash contents.',
        kind: SnackbarKind.ERROR
    });
