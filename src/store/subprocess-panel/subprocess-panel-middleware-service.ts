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
import { OrderDirection, OrderBuilder } from '~/services/api/order-builder';
import { ListResults } from '~/services/common-service/common-service';
import { getSortColumn } from "~/store/data-explorer/data-explorer-reducer";
import { ProcessResource } from '~/models/process';
import { SubprocessPanelColumnNames } from '~/views/subprocess-panel/subprocess-panel-root';
import { FilterBuilder } from '~/services/api/filter-builder';
import { subprocessPanelActions } from './subprocess-panel-actions';
/* import { getProcessStatus } from '../processes/process';
import { ContainerRequestResource } from '~/models/container-request';
import { ContainerResource } from '~/models/container';*/

export class SubprocessMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        try {
            const parentContainerRequestUuid = state.processPanel.containerRequestUuid;
            if (parentContainerRequestUuid === "") { return; }

            const parentContainerRequest = await this.services.containerRequestService.get(parentContainerRequestUuid);
            if (!parentContainerRequest.containerUuid) { return; }

            // Get all the subprocess container requests and containers (not filtered based on the data explorer parameters).
            // This lets us filter based on the combined status of the container request and its container, if it exists.
            let filters = new FilterBuilder().addEqual('requestingContainerUuid', parentContainerRequest.containerUuid).getFilters();
            const containerRequests = await this.services.containerRequestService.list({ filters });
            if (containerRequests.items.length === 0) { return; }
            console.log(containerRequests);

            const containerUuids: string[] = containerRequests.items.reduce((uuids, { containerUuid }) =>
                containerUuid
                    ? [...uuids, containerUuid]
                    : uuids, []);
            filters = new FilterBuilder().addIn('uuid', containerUuids).getFilters();
            // const containers = await this.services.containerService.list({ filters });

            // Find a container requests corresponding container if it exists and check if it should be displayed
            const filteredContainerRequestUuids: string[] = [];
            const filteredContainerUuids: string[] = [];
            /* containerRequests.items.forEach(
                (cr: ContainerRequestResource) => {
                    const c = containers.items.find((c: ContainerResource) => cr.containerUuid === c.uuid);
                    const process = c ? { containerRequest: cr, container: c } : { containerRequest: cr };

                    if (statusFilters === getProcessStatus(process)) {
                        filteredContainerRequestUuids.push(process.containerRequest.uuid);
                        if (process.container) { filteredContainerUuids.push(process.container.uuid); }
                    }
                });
*/
            // Requery with the data expolorer query paramaters to populate the actual user view
            filters = new FilterBuilder().addIn('uuid', filteredContainerRequestUuids).getFilters();
            const containerRequestResources = await this.services.containerRequestService.list({ ...getParams(dataExplorer), filters });
            api.dispatch(updateResources(containerRequestResources.items));

            filters = new FilterBuilder().addIn('uuid', filteredContainerUuids).getFilters();
            const containerResources = await this.services.containerService.list({ ...getParams(dataExplorer), filters });
            api.dispatch(updateResources(containerResources.items));

            api.dispatch(setItems(containerRequestResources));
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
