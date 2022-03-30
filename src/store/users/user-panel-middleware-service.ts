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
            const responseFirstName = await this.services.userService.list(getParamsFirstName(dataExplorer));
            if (responseFirstName.itemsAvailable) {
                api.dispatch(updateResources(responseFirstName.items));
                api.dispatch(setItems(responseFirstName));
            } else {
                const responseLastName = await this.services.userService.list(getParamsLastName(dataExplorer));
                api.dispatch(updateResources(responseLastName.items));
                api.dispatch(setItems(responseLastName));
            }
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

const getParamsFirstName = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFiltersFirstName(dataExplorer)
});

const getParamsLastName = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFiltersLastName(dataExplorer)
});

const getFiltersFirstName = (dataExplorer: DataExplorer) => {
    const filters = new FilterBuilder()
        .addILike("first_name", dataExplorer.searchValue)
        .getFilters();
    return filters;
};

const getFiltersLastName = (dataExplorer: DataExplorer) => {
    const filters = new FilterBuilder()
        .addILike("last_name", dataExplorer.searchValue)
        .getFilters();
    return filters;
};

export const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<UserResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;
        switch (sortColumn.name) {
            case UserPanelColumnNames.NAME:
                order.addOrder(sortDirection, "firstName")
                    .addOrder(sortDirection, "lastName");
                break;
            case UserPanelColumnNames.UUID:
                order.addOrder(sortDirection, "uuid");
                break;
            case UserPanelColumnNames.EMAIL:
                order.addOrder(sortDirection, "email");
                break;
            case UserPanelColumnNames.USERNAME:
                order.addOrder(sortDirection, "username");
                break;
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
