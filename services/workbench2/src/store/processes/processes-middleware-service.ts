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
import { BoundDataExplorerActions } from 'store/data-explorer/data-explorer-action';
import { updateResources } from 'store/resources/resources-actions';
import { ListArguments } from 'services/common-service/common-service';
import { ProcessResource } from 'models/process';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { DataColumns } from 'components/data-table/data-table';
import { ProcessStatusFilter, buildProcessStatusFilters } from '../resource-type-filters/resource-type-filters';
import { ContainerRequestResource, containerRequestFieldsNoMounts } from 'models/container-request';
import { progressIndicatorActions } from '../progress-indicator/progress-indicator-actions';
import { loadMissingProcessesInformation } from '../project-panel/project-panel-middleware-service';

export class ProcessesMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, private actions: BoundDataExplorerActions, id: string) {
        super(id);
    }

    getFilters(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): string | null {
        const columns = dataExplorer.columns as DataColumns<string, ContainerRequestResource>;
        const statusColumnFilters = getDataExplorerColumnFilters(columns, 'Status');
        const activeStatusFilter = Object.keys(statusColumnFilters).find(
            filterName => statusColumnFilters[filterName].selected
        ) || ProcessStatusFilter.ALL;

        const nameFilter = new FilterBuilder().addILike("name", dataExplorer.searchValue).getFilters();
        const statusFilter = buildProcessStatusFilters(new FilterBuilder(), activeStatusFilter).getFilters();

        return joinFilters(
            nameFilter,
            statusFilter,
        );
    }

    getParams(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): ListArguments | null {
        const filters = this.getFilters(api, dataExplorer)
        if (filters === null) {
            return null;
        }
        return {
            ...dataExplorerToListParams(dataExplorer),
            order: getOrder<ProcessResource>(dataExplorer),
            filters
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        try {
            if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

            const params = this.getParams(api, dataExplorer);

            if (params !== null) {
                const containerRequests = await this.services.containerRequestService.list(
                    {
                        ...this.getParams(api, dataExplorer),
                        select: containerRequestFieldsNoMounts
                    });
                api.dispatch(updateResources(containerRequests.items));
                await api.dispatch<any>(loadMissingProcessesInformation(containerRequests.items));
                api.dispatch(this.actions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(containerRequests),
                    items: containerRequests.items.map(resource => resource.uuid),
                }));
            } else {
                api.dispatch(this.actions.SET_ITEMS({
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage,
                    items: [],
                }));
            }
            if (!background) { api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId())); }
        } catch {
            api.dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'Could not fetch process list.',
                kind: SnackbarKind.ERROR
            }));
            if (!background) { api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId())); }
        }
    }
}
