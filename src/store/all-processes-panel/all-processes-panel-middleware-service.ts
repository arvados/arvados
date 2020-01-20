// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataExplorerMiddlewareService } from "~/store/data-explorer/data-explorer-middleware-service";
import { RootState } from "../store";
import { ServiceRepository } from "~/services/services";
import { FilterBuilder } from "~/services/api/filter-builder";
import { allProcessesPanelActions } from "./all-processes-panel-action";
import { Dispatch, MiddlewareAPI } from "redux";
import { resourcesActions } from "~/store/resources/resources-actions";
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { progressIndicatorActions } from '~/store/progress-indicator/progress-indicator-actions.ts';
import { getDataExplorer } from "~/store/data-explorer/data-explorer-reducer";
import { loadMissingProcessesInformation } from "~/store/project-panel/project-panel-middleware-service";

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
                const processItems: any = await this.services.containerRequestService.list({
                    filters: new FilterBuilder()
                        .addILike("name", dataExplorer.searchValue)
                        .getFilters()
                });

                api.dispatch(progressIndicatorActions.PERSIST_STOP_WORKING(this.getId()));
                api.dispatch(resourcesActions.SET_RESOURCES(processItems.items));
                await api.dispatch<any>(loadMissingProcessesInformation(processItems.items));
                api.dispatch(allProcessesPanelActions.SET_ITEMS({
                    items: processItems.items.map((resource: any) => resource.uuid),
                    itemsAvailable: processItems.itemsAvailable,
                    page: Math.floor(processItems.offset / processItems.limit),
                    rowsPerPage: processItems.limit
                }));
            } catch (e) {
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
