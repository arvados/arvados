// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from '~/services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import {
    DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta, getDataExplorerColumnFilters
} from '~/store/data-explorer/data-explorer-middleware-service';
import { RootState } from '~/store/store';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from '~/store/data-explorer/data-explorer-reducer';
import { updateResources } from '~/store/resources/resources-actions';
import { SortDirection } from '~/components/data-table/data-column';
import { OrderDirection, OrderBuilder } from '~/services/api/order-builder';
import { ListResults } from '~/services/common-service/common-service';
import { getSortColumn } from "~/store/data-explorer/data-explorer-reducer";
import { ProcessResource } from '~/models/process';
import { SubprocessPanelColumnNames } from '~/views/subprocess-panel/subprocess-panel-root';
import { FilterBuilder } from '~/services/api/filter-builder';
import { subprocessPanelActions } from './subprocess-panel-actions';
import { DataColumns } from '~/components/data-table/data-table';

export class SubprocessMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const columns = dataExplorer.columns as DataColumns<string>;
        const statusFilters = getDataExplorerColumnFilters(columns, 'Status');
        const activeStatusFilter = Object.keys(statusFilters).find(
            filterName => statusFilters[filterName].selected
        );

        try {
            const parentContainerRequestUuid = state.processPanel.containerRequestUuid;
            if (parentContainerRequestUuid === "") { return; }

            const parentContainerRequest = await this.services.containerRequestService.get(parentContainerRequestUuid);

            if (!parentContainerRequest.containerUuid) { return; }

            // Get all the subprocess' container requests and containers.
            const fb = new FilterBuilder().addEqual('requesting_container_uuid', parentContainerRequest.containerUuid);
            if (activeStatusFilter !== undefined && activeStatusFilter !== 'All') {
                fb.addEqual('container.state', activeStatusFilter);
            }
            const containerRequests = await this.services.containerRequestService.list(
                { ...getParams(dataExplorer), filters: fb.getFilters() });
            if (containerRequests.items.length === 0) { return; }
            const containerUuids: string[] = containerRequests.items.reduce(
                (uuids, { containerUuid }) =>
                    containerUuid
                        ? [...uuids, containerUuid]
                        : uuids, []);
            const containers = await this.services.containerService.list({
                filters: new FilterBuilder().addIn('uuid', containerUuids).getFilters()
            });

            // Populate the actual user view
            api.dispatch(updateResources(containerRequests.items));
            api.dispatch(updateResources(containers.items));
            api.dispatch(setItems(containerRequests));
        } catch {
            api.dispatch(couldNotFetchSubprocesses());
        }
    }
}

export const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer)
});

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<ProcessResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === SubprocessPanelColumnNames.NAME ? "name" : "modifiedAt";
        return order
            .addOrder(sortDirection, columnName)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

export const setItems = (listResults: ListResults<ProcessResource>) =>
    subprocessPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchSubprocesses = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch subprocesses.',
        kind: SnackbarKind.ERROR
    });
