// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorerMiddlewareService, listResultsToDataExplorerItemsMeta, dataExplorerToListParams } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer, getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { GroupsPanelActions } from 'store/groups-panel/groups-panel-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { updateResources } from 'store/resources/resources-actions';
import { OrderBuilder, OrderDirection } from 'services/api/order-builder';
import { GroupResource, GroupClass } from 'models/group';
import { SortDirection } from 'components/data-table/data-column';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { ListArguments, ListResults } from "services/common-service/common-service";
import { couldNotFetchItemsAvailable } from "store/data-explorer/data-explorer-action";

export class GroupsPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    getOrder = (dataExplorer: DataExplorer) => {
        const sortColumn = getSortColumn<GroupResource>(dataExplorer);
        const order = new OrderBuilder<GroupResource>();
        if (sortColumn && sortColumn.sort) {
            const sortDirection = sortColumn.sort.direction === SortDirection.ASC ? OrderDirection.ASC : OrderDirection.DESC;

            // Use createdAt as a secondary sort column so we break ties consistently.
            return order
                .addOrder(sortDirection, sortColumn.sort.field)
                .addOrder(OrderDirection.DESC, "createdAt")
                .getOrder();
        } else {
            return order.getOrder();
        }
    };

    getFilters(dataExplorer: DataExplorer): string {
        return new FilterBuilder()
            .addEqual('group_class', GroupClass.ROLE)
            .addILike('name', dataExplorer.searchValue)
            .getFilters();
    }

    getParams(dataExplorer: DataExplorer): ListArguments {
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters: this.getFilters(dataExplorer),
            order: this.getOrder(dataExplorer),
            count: 'none',
        };
    }

    getCountParams(dataExplorer: DataExplorer): ListArguments {
        return {
            filters: this.getFilters(dataExplorer),
            limit: 0,
            count: 'exact',
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(groupsPanelDataExplorerIsNotSet());
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                // Get items
                const groups = await this.services.groupsService.list(this.getParams(dataExplorer));
                api.dispatch(updateResources(groups.items));
                api.dispatch(GroupsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(groups),
                    items: groups.items.map(resource => resource.uuid),
                }));

                // Get group member counts
                groups.items.map(group => (
                    this.services.permissionService.list({
                        limit: 0,
                        filters: new FilterBuilder()
                            .addEqual('head_uuid', group.uuid)
                            .getFilters()
                    }).then(members => {
                        api.dispatch(updateResources([{
                            ...group,
                            memberCount: members.itemsAvailable,
                        } as GroupResource]));
                    }).catch(e => {
                        // In case of error, store null to stop spinners and show failure icon
                        api.dispatch(updateResources([{
                            ...group,
                            memberCount: null,
                        } as GroupResource]));
                    })
                ));
            } catch (e) {
                api.dispatch(couldNotFetchGroupList());
            } finally {
                api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
            }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        if (criteriaChanged) {
            // Get itemsAvailable
            return this.services.groupsService.list(this.getCountParams(dataExplorer))
                .then((results: ListResults<GroupResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(GroupsPanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
        }
    }
}

const groupsPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Groups panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchGroupList = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch groups.',
        kind: SnackbarKind.ERROR
    });
