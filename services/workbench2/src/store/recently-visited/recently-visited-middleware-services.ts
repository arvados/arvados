// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import {
    DataExplorerMiddlewareService,
    dataExplorerToListParams,
} from "../data-explorer/data-explorer-middleware-service";
import { ServiceRepository } from "services/services";
import { MiddlewareAPI, Dispatch } from "redux";
import { RootState } from 'store/store';
import { getDataExplorer, DataExplorer } from 'store/data-explorer/data-explorer-reducer';
import { updateResources } from 'store/resources/resources-actions';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { ContentsArguments } from 'services/groups-service/groups-service';
import { FilterBuilder } from 'services/api/filter-builder';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { RecentUuid } from 'models/user';

export class RecentlyVisitedMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
            const response = await this.services.groupsService
                .contents('', getParams(dataExplorer, state.auth.user?.prefs?.wb?.recentUuids || []));
            api.dispatch(updateResources(response.items));
            if (response.included) { api.dispatch(updateResources(response.included)); }
        } catch (e) {
            api.dispatch(couldNotFetchRecentlyVisited());
        } finally {
            api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
        }
    }

    // required by DataExplorerMiddlewareService, but not used
    async requestCount() {}
}

const getParams = (dataExplorer: DataExplorer, recents: RecentUuid[]): ContentsArguments => ({
    ...dataExplorerToListParams(dataExplorer),
    filters: new FilterBuilder().addIn('uuid', recents.map(recent => recent.uuid)).getFilters(),
    include: ["owner_uuid", "container_uuid"]
});

const couldNotFetchRecentlyVisited = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch recently visited items.',
        kind: SnackbarKind.ERROR
    });
