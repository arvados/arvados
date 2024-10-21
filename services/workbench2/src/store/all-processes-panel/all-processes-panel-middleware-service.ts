// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getDataExplorerColumnFilters } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { ServiceRepository } from "services/services";
import { joinFilters } from "services/api/filter-builder";
import { allProcessesPanelActions } from "./all-processes-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorer } from "store/data-explorer/data-explorer-reducer";
import { DataColumns } from "components/data-table/data-column";
import {
    serializeOnlyProcessTypeFilters
} from "../resource-type-filters/resource-type-filters";
import { AllProcessesPanelColumnNames } from "views/all-processes-panel/all-processes-panel";
import { ProcessesMiddlewareService } from "store/processes/processes-middleware-service";
import { ContainerRequestResource } from 'models/container-request';

export class AllProcessesPanelMiddlewareService extends ProcessesMiddlewareService {
    constructor(services: ServiceRepository, id: string) {
        super(services, allProcessesPanelActions, id);
    }

    getFilters(api: MiddlewareAPI<Dispatch, RootState>, dataExplorer: DataExplorer): string | null {
        const sup = super.getFilters(api, dataExplorer);
        if (sup === null) { return null; }
        const columns = dataExplorer.columns as DataColumns<string, ContainerRequestResource>;

        const typeFilters = serializeOnlyProcessTypeFilters(true)(getDataExplorerColumnFilters(columns, AllProcessesPanelColumnNames.TYPE));
        return joinFilters(sup, typeFilters);
    }
}
