// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService, dataExplorerToListParams, getDataExplorerColumnFilters } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { ServiceRepository } from "services/services";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { allProcessesPanelActions } from "./all-processes-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { getDataExplorer, DataExplorer, getSortColumn } from "store/data-explorer/data-explorer-reducer";
import { loadMissingProcessesInformation } from "store/project-panel/project-panel-middleware-service";
import { DataColumns } from "components/data-table/data-table";
import {
    ProcessStatusFilter,
    buildProcessStatusFilters,
    serializeOnlyProcessTypeFilters
} from "../resource-type-filters/resource-type-filters";
import { AllProcessesPanelColumnNames } from "views/all-processes-panel/all-processes-panel";
import { OrderBuilder, OrderDirection } from "services/api/order-builder";
import { ProcessResource } from "models/process";
import { SortDirection } from "components/data-table/data-column";

export class AllProcessesPanelMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        if (!dataExplorer) {
            api.dispatch(allProcessesPanelDataExplorerIsNotSet());
        } else {
            try {
                api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));
                const processItems = await this.services.containerRequestService.list(
                    { ...getParams(dataExplorer) });

                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(resourcesActions.SET_RESOURCES(processItems.items));
                await api.dispatch<any>(loadMissingProcessesInformation(processItems.items));
                api.dispatch(allProcessesPanelActions.SET_ITEMS({
                    items: processItems.items.map((resource: any) => resource.uuid),
                    itemsAvailable: processItems.itemsAvailable,
                    page: Math.floor(processItems.offset / processItems.limit) || 0,
                    rowsPerPage: processItems.limit
                }));
            } catch {
                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(allProcessesPanelActions.SET_ITEMS({
                    items: [],
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage
                }));
                api.dispatch(couldNotFetchAllProcessesListing());
            }
        }
    }
}

const getParams = ( dataExplorer: DataExplorer ) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder(dataExplorer),
    filters: getFilters(dataExplorer)
});

const getFilters = ( dataExplorer: DataExplorer ) => {
    const columns = dataExplorer.columns as DataColumns<string>;
    const statusColumnFilters = getDataExplorerColumnFilters(columns, 'Status');
    const activeStatusFilter = Object.keys(statusColumnFilters).find(
        filterName => statusColumnFilters[filterName].selected
    ) || ProcessStatusFilter.ALL;

    const nameFilter = new FilterBuilder().addILike("name", dataExplorer.searchValue).getFilters();
    const statusFilter = buildProcessStatusFilters(new FilterBuilder(), activeStatusFilter).getFilters();
    const typeFilters = serializeOnlyProcessTypeFilters(getDataExplorerColumnFilters(columns, AllProcessesPanelColumnNames.TYPE));

    return joinFilters(
        nameFilter,
        statusFilter,
        typeFilters
    );
};

const getOrder = (dataExplorer: DataExplorer) => {
    const sortColumn = getSortColumn(dataExplorer);
    const order = new OrderBuilder<ProcessResource>();
    if (sortColumn) {
        const sortDirection = sortColumn && sortColumn.sortDirection === SortDirection.ASC
            ? OrderDirection.ASC
            : OrderDirection.DESC;

        const columnName = sortColumn && sortColumn.name === AllProcessesPanelColumnNames.NAME ? "name" : "createdAt";
        return order
            .addOrder(sortDirection, columnName)
            .getOrder();
    } else {
        return order.getOrder();
    }
};

const allProcessesPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'All Processes panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchAllProcessesListing = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch All Processes listing.',
        kind: SnackbarKind.ERROR
    });
