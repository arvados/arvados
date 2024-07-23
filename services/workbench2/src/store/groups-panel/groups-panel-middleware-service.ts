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
import { ListArguments } from "services/common-service/common-service";

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
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(groupsPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const groups = await this.services.groupsService.list(this.getParams(dataExplorer));
                api.dispatch(updateResources(groups.items));
                api.dispatch(GroupsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(groups),
                    items: groups.items.map(item => item.uuid),
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
