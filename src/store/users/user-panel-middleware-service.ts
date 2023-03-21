// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateResources } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { SortDirection } from 'components/data-table/data-column';
import { OrderDirection, OrderBuilder } from 'services/api/order-builder';
import { ListResults } from 'services/common-service/common-service';
import { userBindedActions } from 'store/users/users-actions';
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { UserResource } from 'models/user';
import { UserPanelColumnNames } from 'views/user-panel/user-panel';
import { BuiltinGroups, getBuiltinGroupUuid } from 'models/group';
import { LinkClass } from 'models/link';

export class UserMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            const users = await this.services.userService.list(getParams(dataExplorer));
            api.dispatch(updateResources(users.items));
            api.dispatch(setItems(users));

            // Get "all users" group memberships
            const allUsersGroupUuid = getBuiltinGroupUuid(state.auth.localCluster, BuiltinGroups.ALL);
            const allUserMemberships = await this.services.permissionService.list({
                filters: new FilterBuilder()
                    .addEqual('head_uuid', allUsersGroupUuid)
                    .addEqual('link_class', LinkClass.PERMISSION)
                    .getFilters()
            });
            api.dispatch(updateResources(allUserMemberships.items));
        } catch {
            api.dispatch(couldNotFetchUsers());
        }
    }
}

const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: new FilterBuilder()
        .addFullTextSearch(dataExplorer.searchValue)
        .getFilters()
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn<UserResource>(dataExplorer);
    const order = new OrderBuilder<UserResource>();
    if (sortColumn && sortColumn.sort) {
        const sortDirection = sortColumn.sort.direction === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        if (sortColumn.name === UserPanelColumnNames.NAME) {
            order.addOrder(sortDirection, "firstName")
                .addOrder(sortDirection, "lastName");
        } else {
            order.addOrder(sortDirection, sortColumn.sort.field);
        }
    }
    return order.getOrder();
};

export const setItems = (listResults: ListResults<UserResource>) =>
    userBindedActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchUsers = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch users.',
        kind: SnackbarKind.ERROR
    });
