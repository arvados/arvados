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
import { getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { computeNodesActions } from 'store/compute-nodes/compute-nodes-actions';
import { OrderDirection, OrderBuilder } from 'services/api/order-builder';
import { ListResults } from 'services/common-service/common-service';
import { NodeResource } from 'models/node';
import { SortDirection } from 'components/data-table/data-column';
import { ComputeNodePanelColumnNames } from 'views/compute-node-panel/compute-node-panel-root';

export class ComputeNodeMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            const response = await this.services.nodeService.list(getParams(dataExplorer));
            api.dispatch(updateResources(response.items));
            api.dispatch(setItems(response));
        } catch {
            api.dispatch(couldNotFetchLinks());
        }
    }
}

export const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer)
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<NodeResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === ComputeNodePanelColumnNames.UUID ? "uuid" : "modifiedAt";
        return order
            .addOrder(sortDirection, columnName)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

export const setItems = (listResults: ListResults<NodeResource>) =>
    computeNodesActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchLinks = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch compute nodes.',
        kind: SnackbarKind.ERROR
    });
