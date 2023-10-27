// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import {
    dataExplorerToListParams, listResultsToDataExplorerItemsMeta, getDataExplorerColumnFilters, getOrder
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
import { ProcessesMiddlewareService } from "store/processes/processes-middleware-service";
import { getProcess } from "store/processes/process";

export class SubprocessMiddlewareService extends ProcessesMiddlewareService {
    constructor(services: ServiceRepository, id: string) {
        super(services, subprocessPanelActions, id);
    }

    getFilters(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): string | null {
        const state = api.getState();
        const parentContainerRequestUuid = state.processPanel.containerRequestUuid;
        if (!parentContainerRequestUuid) { return null; }

        const process = getProcess(parentContainerRequestUuid)(state.resources);
        if (!process?.container) { return null; }

        const requesting_container = new FilterBuilder().addEqual('requesting_container_uuid', process.container.uuid).getFilters();
        const sup = super.getFilters(api, dataExplorer);
        if (sup === null) { return null; }

        return joinFilters(sup, requesting_container);
    }
}
