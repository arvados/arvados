// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorerMiddlewareService, listResultsToDataExplorerItemsMeta, dataExplorerToListParams } from "~/store/data-explorer/data-explorer-middleware-service";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getDataExplorer } from "~/store/data-explorer/data-explorer-reducer";
import { GroupsPanelActions } from '~/store/groups-panel/groups-panel-actions';
import { FilterBuilder } from '~/services/api/filter-builder';
import { updateResources } from '~/store/resources/resources-actions';

export class GroupsPanelMiddlewareService extends DataExplorerMiddlewareService {

    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {

        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());

        if (!dataExplorer) {

            api.dispatch(groupsPanelDataExplorerIsNotSet());

        } else {

            try {

                const filters = new FilterBuilder()
                    .addEqual('groupClass', null)
                    .getFilters();

                const response = await this.services.groupsService
                    .list({
                        ...dataExplorerToListParams(dataExplorer),
                        filters,
                    });

                api.dispatch(updateResources(response.items));

                api.dispatch(GroupsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(response),
                    items: response.items.map(item => item.uuid),
                }));


            } catch (e) {

                api.dispatch(couldNotFetchFavoritesContents());

            }
        }
    }
}

const groupsPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Groups panel is not ready.'
    });

const couldNotFetchFavoritesContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch groups.',
        kind: SnackbarKind.ERROR
    });
