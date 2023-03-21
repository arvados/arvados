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
import { ListResults } from 'services/common-service/common-service';
import { LinkResource } from 'models/link';
import { linkPanelActions } from 'store/link-panel/link-panel-actions';

export class LinkMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            const response = await this.services.linkService.list(getParams(dataExplorer));
            api.dispatch(updateResources(response.items));
            api.dispatch(setItems(response));
        } catch {
            api.dispatch(couldNotFetchLinks());
        }
    }
}

export const getParams = (dataExplorer: DataExplorer) => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder<LinkResource>(dataExplorer)
});

export const setItems = (listResults: ListResults<LinkResource>) =>
    linkPanelActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchLinks = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch links.',
        kind: SnackbarKind.ERROR
    });
