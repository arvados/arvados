// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, dataExplorerToListParams, getOrder, listResultsToDataExplorerItemsMeta } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateResources } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { WorkflowResource } from 'models/workflow';
import { ListResults } from 'services/common-service/common-service';
import { workflowPanelActions } from 'store/workflow-panel/workflow-panel-actions';
import { matchRegisteredWorkflowRoute } from 'routes/routes';
import { ProcessesMiddlewareService } from "store/processes/processes-middleware-service";
import { workflowProcessesPanelActions } from "./workflow-panel-actions";
import { joinFilters } from "services/api/filter-builder";

export class WorkflowMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            const response = await this.services.workflowService.list(getParams(dataExplorer));
            api.dispatch(updateResources(response.items));
            api.dispatch(setItems(response));
        } catch {
            api.dispatch(couldNotFetchWorkflows());
        }
    }

    // Don't use separate request count on unused WF panel
    async requestCount() {}
}

export const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder<WorkflowResource>(dataExplorer),
    filters: getFilters(dataExplorer)
});

export const getFilters = (dataExplorer: DataExplorer) => {
    const filters = new FilterBuilder()
        .addILike("name", dataExplorer.searchValue)
        .getFilters();
    return filters;
};

export const setItems = (listResults: ListResults<WorkflowResource>) =>
    workflowPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchWorkflows = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch workflows.',
        kind: SnackbarKind.ERROR
    });


export class WorkflowProcessesMiddlewareService extends ProcessesMiddlewareService {
    constructor(services: ServiceRepository, id: string) {
        super(services, workflowProcessesPanelActions, id);
    }

    getFilters(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): string | null {
        const state = api.getState();

        if (!state.router.location) { return null; }

        const registeredWorkflowMatch = matchRegisteredWorkflowRoute(state.router.location.pathname);
        if (!registeredWorkflowMatch) { return null; }

        const workflow_uuid = registeredWorkflowMatch.params.id;

        const requesting_container = new FilterBuilder().addEqual('properties.template_uuid', workflow_uuid).getFilters();
        const sup = super.getFilters(api, dataExplorer);
        if (sup === null) { return null; }

        return joinFilters(sup, requesting_container);
    }
}
