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

export class SubprocessMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        try {
            const crUuid = state.processPanel.containerRequestUuid;
            if (crUuid !== "") {
                const containerRequest = await this.services.containerRequestService.get(crUuid);
                if (containerRequest.containerUuid) {
                    const filters = new FilterBuilder().addEqual('requestingContainerUuid', containerRequest.containerUuid).getFilters();
                    const containerRequests = await this.services.containerRequestService.list({ ...getParams(dataExplorer), filters });
                    api.dispatch(updateResources(containerRequests.items));
                    api.dispatch(setItems(containerRequests));

                    const containerUuids: string[] = containerRequests.items.reduce((uuids, { containerUuid }) =>
                        containerUuid
                            ? [...uuids, containerUuid]
                            : uuids, []);

                    if (containerUuids.length > 0) {
                        const filters = new FilterBuilder().addIn('uuid', containerUuids).getFilters();
                        const containers = await this.services.containerService.list({ filters });
                        api.dispatch<any>(updateResources(containers.items));
                    }
                }
            }
            // TODO: set filters based on process panel state

        } catch {
            api.dispatch(couldNotFetchSubprocesses());
        }
    }
}

/*export const getFilters = (processPanel: ProcessPanelState, processes: Process[]) => {
    const grouppedProcesses = groupBy(processes, getProcessStatus);
    return Object
        .keys(processPanel.filters)
        .map(filter => ({
            label: filter,
            value: (grouppedProcesses[filter] || []).length,
            checked: processPanel.filters[filter],
            key: filter,
        }));
    };
*/

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
