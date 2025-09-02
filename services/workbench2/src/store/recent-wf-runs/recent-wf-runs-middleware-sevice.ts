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
import { ContentsArguments } from 'services/groups-service/groups-service';
import { FilterBuilder } from 'services/api/filter-builder';
import { containerRequestFieldsNoMounts } from 'models/container-request';
import { progressIndicatorActions } from '../progress-indicator/progress-indicator-actions';
import { containerFieldsNoMounts } from 'store/processes/processes-actions';
import { recentWorkflowRunsActions } from './recent-wf-runs-action';

export class RecentWorkflowsMiddlewareService extends DataExplorerMiddlewareService {
        constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    getParams(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): ContentsArguments | null {
        return {
            ...dataExplorerToListParams(dataExplorer),
            filters: new FilterBuilder()
                .addIsA('uuid', 'arvados#containerRequest')
                .addEqual('container_requests.requesting_container_uuid', null)
                .getFilters(),
            select: containerRequestFieldsNoMounts,
            count: 'none',
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
                        select: [...containerRequestFieldsNoMounts, "can_write", "can_manage"].concat(containerFieldsNoMounts)
                });
                api.dispatch(updateResources(containerRequests.items));
                if (containerRequests.included) {
                    api.dispatch(updateResources(containerRequests.included));
                }

                api.dispatch(recentWorkflowRunsActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(containerRequests),
                    items: containerRequests.items.map(resource => resource.uuid),
                }));
            } else {
                api.dispatch(recentWorkflowRunsActions.SET_ITEMS({
                    itemsAvailable: 0,
                    page: 0,
                    rowsPerPage: dataExplorer.rowsPerPage,
                    items: [],
                }));
            }
        } catch {
            api.dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'Could not fetch recent workflow runs.',
                kind: SnackbarKind.ERROR
            }));
        } finally {
            if (!background) { api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId())); }
        }
    }

    async requestCount() {}
}
