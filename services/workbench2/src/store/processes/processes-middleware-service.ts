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
import { BoundDataExplorerActions, couldNotFetchItemsAvailable } from 'store/data-explorer/data-explorer-action';
import { updateResources } from 'store/resources/resources-actions';
import { ListArguments, ListResults } from 'services/common-service/common-service';
import { ContentsArguments } from 'services/groups-service/groups-service';
import { ProcessResource } from 'models/process';
import { FilterBuilder, joinFilters } from 'services/api/filter-builder';
import { DataColumns } from 'components/data-table/data-table';
import { ProcessStatusFilter, buildProcessStatusFilters } from '../resource-type-filters/resource-type-filters';
import { ContainerRequestResource, containerRequestFieldsNoMounts } from 'models/container-request';
import { progressIndicatorActions } from '../progress-indicator/progress-indicator-actions';
import { loadMissingProcessesInformation } from '../project-panel/project-panel-run-middleware-service';
import { containerFieldsNoMounts } from 'store/processes/processes-actions';

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


        let filters = new FilterBuilder().addIsA('uuid', 'arvados#containerRequest');
        if (dataExplorer.searchValue && dataExplorer.searchValue !== "") {
            filters = filters.addILike("name", dataExplorer.searchValue);
        }

        return buildProcessStatusFilters(filters, activeStatusFilter).getFilters();
    }


    getParams(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): ContentsArguments | null {
        const filters = this.getFilters(api, dataExplorer)
        if (filters === null) {
            return null;
        }
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters,
            order: getOrder<ProcessResource>(dataExplorer),
            select: containerRequestFieldsNoMounts,
            count: 'none',
        };
    }

    getCountParams(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): ListArguments | null {
        const filters = this.getFilters(api, dataExplorer);
        if (filters === null) {
            return null;
        }
        return {
            filters,
            limit: 0,
            count: 'exact',
            include: ["owner_uuid", "container_uuid"]
        };
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());

        try {
            if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
            const params = this.getParams(api, dataExplorer);

            // Get items
            if (params !== null) {
                const containerRequests = await this.services.groupsService.contents('',
                    {
                        ...this.getParams(api, dataExplorer),
                        select: containerRequestFieldsNoMounts.concat(containerFieldsNoMounts)
                });
                api.dispatch(updateResources(containerRequests.items));
                if (containerRequests.included) {
                    api.dispatch(updateResources(containerRequests.included));
                }

                // This is the one
                //await api.dispatch<any>(loadMissingProcessesInformation(containerRequests.items));

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
        } catch {
            api.dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'Could not fetch process list.',
                kind: SnackbarKind.ERROR
            }));
        } finally {
            if (!background) { api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId())); }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        const countParams = this.getCountParams(api, dataExplorer);

        if (criteriaChanged && countParams !== null) {
            // Get itemsAvailable
            return this.services.containerRequestService.list(countParams)
                .then((results: ListResults<ContainerRequestResource>) => {
                    console.log(results);
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(this.actions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
        }
    }
}
