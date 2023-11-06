// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import {
    DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta, getDataExplorerColumnFilters, getOrder
} from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateResources } from 'store/resources/resources-actions';
import { ListResults } from 'services/common-service/common-service';
import { ProcessResource } from 'models/process';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { subprocessPanelActions } from './subprocess-panel-actions';
import { DataColumns } from 'components/data-table/data-table';
import { ProcessStatusFilter, buildProcessStatusFilters } from '../resource-type-filters/resource-type-filters';
import { ContainerRequestResource, containerRequestFieldsNoMounts } from 'models/container-request';
import { progressIndicatorActions } from '../progress-indicator/progress-indicator-actions';
import { loadMissingProcessesInformation } from '../project-panel/project-panel-middleware-service';

export class SubprocessMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const parentContainerRequestUuid = state.processPanel.containerRequestUuid;
        if (parentContainerRequestUuid === "") { return; }
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        try {
            if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
            const parentContainerRequest = await this.services.containerRequestService.get(parentContainerRequestUuid);
            if (parentContainerRequest.containerUuid) {
                const containerRequests = await this.services.containerRequestService.list(
                    {
                        ...getParams(dataExplorer, parentContainerRequest),
                        select: containerRequestFieldsNoMounts
                    });
                api.dispatch(updateResources(containerRequests.items));
                await api.dispatch<any>(loadMissingProcessesInformation(containerRequests.items));
                // Populate the actual user view
                api.dispatch(setItems(containerRequests));
            }
            if (!background) { api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId())); }
        } catch {
            if (!background) { api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId())); }
            api.dispatch(couldNotFetchSubprocesses());
        }
    }
}

export const getParams = (
    dataExplorer: DataExplorer,
    parentContainerRequest: ContainerRequestResource) => ({
        ...dataExplorerToListParams(dataExplorer),
        order: getOrder<ProcessResource>(dataExplorer),
        filters: getFilters(dataExplorer, parentContainerRequest)
    });

export const getFilters = (
    dataExplorer: DataExplorer,
    parentContainerRequest: ContainerRequestResource) => {
    const columns = dataExplorer.columns as DataColumns<string, ProcessResource>;
    const statusColumnFilters = getDataExplorerColumnFilters(columns, 'Status');
    const activeStatusFilter = Object.keys(statusColumnFilters).find(
        filterName => statusColumnFilters[filterName].selected
    ) || ProcessStatusFilter.ALL;

    // Get all the subprocess' container requests and containers.
    const fb = new FilterBuilder().addEqual('requesting_container_uuid', parentContainerRequest.containerUuid);
    const statusFilters = buildProcessStatusFilters(fb, activeStatusFilter).getFilters();

    const nameFilters = dataExplorer.searchValue
        ? new FilterBuilder()
            .addILike("name", dataExplorer.searchValue)
            .getFilters()
        : '';

    return joinFilters(
        nameFilters,
        statusFilters
    );
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
