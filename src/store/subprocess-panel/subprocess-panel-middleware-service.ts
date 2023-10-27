// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from "../store";
import { ServiceRepository } from "services/services";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorer } from "store/data-explorer/data-explorer-reducer";
import { ProcessesMiddlewareService } from "store/processes/processes-middleware-service";
import { subprocessPanelActions } from './subprocess-panel-actions';
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
