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
import { ListArguments, ListResults } from 'services/common-service/common-service';
import { ExternalCredential } from 'models/external-credential';
import { externalCredentialsActions } from 'store/external-credentials/external-credentials-actions';
import { couldNotFetchItemsAvailable } from 'store/data-explorer/data-explorer-action';

export class ExternalCredentialsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const dataExplorer = getDataExplorer(state.dataExplorer, this.getId());
        try {
            const response = await this.services.externalCredentialsService.list(getParams(dataExplorer));
            api.dispatch(updateResources(response.items));
            api.dispatch(setItems(response));
        } catch {
            api.dispatch(couldNotFetchExternalCredentials());
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        if (criteriaChanged) {
            // Get itemsAvailable
            return this.services.externalCredentialsService.list(getCountParams())
                .then((results: ListResults<ExternalCredential>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(externalCredentialsActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
        }
    }
}

const getParams = (dataExplorer: DataExplorer): ListArguments => ({
    ...dataExplorerToListParams(dataExplorer),
    order: getOrder<ExternalCredential>(dataExplorer),
    count: 'none',
});

const getCountParams = (): ListArguments => ({
    limit: 0,
    count: 'exact',
});

export const setItems = (listResults: ListResults<ExternalCredential>) =>
    externalCredentialsActions.SET_ITEMS({
        ...listResultsToDataExplorerItemsMeta(listResults),
        items: listResults.items.map(resource => resource.uuid),
    });

const couldNotFetchExternalCredentials = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch external credentials.',
        kind: SnackbarKind.ERROR
    });
