// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorerMiddlewareService, listResultsToDataExplorerItemsMeta, dataExplorerToListParams } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getDataExplorer, getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { GroupsPanelActions } from 'store/groups-panel/groups-panel-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { updateResources } from 'store/resources/resources-actions';
import { OrderBuilder, OrderDirection } from 'services/api/order-builder';
import { GroupResource, GroupClass } from 'models/group';
import { SortDirection } from 'components/data-table/data-column';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export class GroupsPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }
    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(groupsPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const sortColumn = getSortColumn<GroupResource>(dataExplorer);
                const order = new OrderBuilder<GroupResource>();
                if (sortColumn && sortColumn.sort) {
                    const direction =
                        sortColumn.sort.direction === SortDirection.ASC
                            ? OrderDirection.ASC
                            : OrderDirection.DESC;
                    order.addOrder(direction, sortColumn.sort.field);
                }
                const filters = new FilterBuilder()
                    .addEqual('group_class', GroupClass.ROLE)
                    .addILike('name', dataExplorer.searchValue)
                    .getFilters();
                const response = await this.services.groupsService
                    .list({
                        ...dataExplorerToListParams(dataExplorer),
                        filters,
                        order: order.getOrder(),
                    });
                api.dispatch(updateResources(response.items));
                api.dispatch(GroupsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(response),
                    items: response.items.map(item => item.uuid),
                }));
                const permissions = await this.services.permissionService.list({
                    filters: new FilterBuilder()
                        .addIn('head_uuid', response.items.map(item => item.uuid))
                        .getFilters()
                });
                api.dispatch(updateResources(permissions.items));
            } catch (e) {
                api.dispatch(couldNotFetchFavoritesContents());
            } finally {
                api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
            }
        }
    }
}

const groupsPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Groups panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchFavoritesContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch groups.',
        kind: SnackbarKind.ERROR
    });
