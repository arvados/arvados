// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { dataExplorerToListParams, getDataExplorerColumnFilters, getOrder } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { ServiceRepository } from "services/services";
import { FilterBuilder, joinFilters } from "services/api/filter-builder";
import { allProcessesPanelActions } from "./all-processes-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { getDataExplorer, DataExplorer } from "store/data-explorer/data-explorer-reducer";
import { loadMissingProcessesInformation } from "store/project-panel/project-panel-middleware-service";
import { DataColumns } from "components/data-table/data-table";
import {
    ProcessStatusFilter,
    buildProcessStatusFilters,
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

        const typeFilters = serializeOnlyProcessTypeFilters(getDataExplorerColumnFilters(columns, AllProcessesPanelColumnNames.TYPE));
        return joinFilters(sup, typeFilters);
    }
}
