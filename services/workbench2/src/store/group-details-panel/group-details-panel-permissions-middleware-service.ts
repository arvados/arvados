// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorerMiddlewareService, dataExplorerToListParams, listResultsToDataExplorerItemsMeta } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { DataExplorer, getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { FilterBuilder } from 'services/api/filter-builder';
import { updateResources } from 'store/resources/resources-actions';
import { getCurrentGroupDetailsPanelUuid, GroupPermissionsPanelActions } from 'store/group-details-panel/group-details-panel-actions';
import { LinkClass } from 'models/link';
import { ResourceKind } from 'models/resource';
import { ListArguments, ListResults } from "services/common-service/common-service";
import { PermissionResource } from "models/permission";
import { couldNotFetchItemsAvailable } from "store/data-explorer/data-explorer-action";
import { ProjectResource } from "models/project";
import { CollectionResource } from "models/collection";
import { UserResource } from "models/user";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export class GroupDetailsPanelPermissionsMiddlewareService extends DataExplorerMiddlewareService {

    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const groupUuid = getCurrentGroupDetailsPanelUuid(api.getState().properties);
        if (!dataExplorer || !groupUuid) {
            // No-op if data explorer is not set since refresh may be triggered from elsewhere
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }

                // Get items
                const permissionsOut = await this.services.permissionService.list(getParams(dataExplorer, groupUuid));
                api.dispatch(updateResources(permissionsOut.items));

                api.dispatch(GroupPermissionsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(permissionsOut),
                    items: permissionsOut.items,
                }));

                const userUuids = permissionsOut.items
                    .filter((item) => item.headKind === ResourceKind.USER)
                    .map(item => item.headUuid);
                if (userUuids.length) {
                    this.services.userService
                        .list(getMetadataParams(dataExplorer, userUuids))
                        .then((usersOut: ListResults<UserResource>) => (
                            api.dispatch(updateResources(usersOut.items))
                        ));
                }

                const collectionUuids = permissionsOut.items
                    .filter((item) => item.headKind === ResourceKind.COLLECTION)
                    .map(item => item.headUuid);
                if (collectionUuids.length) {
                    this.services.collectionService
                        .list(getMetadataParams(dataExplorer, collectionUuids))
                        .then((collectionsOut: ListResults<CollectionResource>) => (
                            api.dispatch(updateResources(collectionsOut.items))
                        ));
                }

                const projectUuids = permissionsOut.items
                    .filter((item) => item.headKind === ResourceKind.PROJECT)
                    .map(item => item.headUuid);
                if (projectUuids.length) {
                    this.services.projectService
                        .list(getMetadataParams(dataExplorer, projectUuids))
                        .then((projectsOut: ListResults<ProjectResource>) => (
                            api.dispatch(updateResources(projectsOut.items))
                        ));
                }
            } catch (e) {
                api.dispatch(couldNotFetchGroupDetailsContents());
            } finally {
                api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
            }
        }
    }

    async requestCount(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const state = api.getState();
        const groupUuid = getCurrentGroupDetailsPanelUuid(state.properties);

        if (criteriaChanged && groupUuid) {
            // Get itemsAvailable
            return this.services.permissionService.list(getCountParams(groupUuid))
                .then((results: ListResults<PermissionResource>) => {
                    if (results.itemsAvailable !== undefined) {
                        api.dispatch<any>(GroupPermissionsPanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
                    } else {
                        couldNotFetchItemsAvailable();
                    }
                });
        }
    }
}

export const getParams = (dataExplorer: DataExplorer, groupUuid: string): ListArguments => ({
    ...dataExplorerToListParams(dataExplorer),
    filters: getFilters(groupUuid),
    count: 'none',
});

export const getMetadataParams = (dataExplorer: DataExplorer, uuids: string[]): ListArguments => ({
    limit: dataExplorer.rowsPerPage,
    filters: new FilterBuilder()
        .addIn('uuid', uuids)
        .getFilters(),
    count: 'none',
});

export const getCountParams = (groupUuid: string): ListArguments => ({
    filters: getFilters(groupUuid),
    limit: 0,
    count: 'exact',
});

export const getFilters = (groupUuid: string) => {
    return new FilterBuilder()
        .addEqual('tail_uuid', groupUuid)
        .addEqual('link_class', LinkClass.PERMISSION)
        .getFilters();
};

const couldNotFetchGroupDetailsContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch group permissions.',
        kind: SnackbarKind.ERROR
    });
