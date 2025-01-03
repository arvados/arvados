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
import { getCurrentGroupDetailsPanelUuid, GroupMembersPanelActions } from 'store/group-details-panel/group-details-panel-actions';
import { LinkClass } from 'models/link';
import { ResourceKind } from 'models/resource';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';
import { couldNotFetchItemsAvailable } from "store/data-explorer/data-explorer-action";
import { ListArguments, ListResults } from "services/common-service/common-service";
import { PermissionResource } from "models/permission";
import { UserResource } from "models/user";
import { ProjectResource } from "models/project";

export class GroupDetailsPanelMembersMiddlewareService extends DataExplorerMiddlewareService {

    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>, criteriaChanged?: boolean, background?: boolean) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const groupUuid = getCurrentGroupDetailsPanelUuid(api.getState().properties);
        if (!dataExplorer || !groupUuid) {
            // Noop if data explorer refresh is triggered from another panel
            return;
        } else {
            try {
                if (!background) { api.dispatch(progressIndicatorActions.START_WORKING(this.getId())); }
                const groupResource = await this.services.groupsService.get(groupUuid);
                api.dispatch(updateResources([groupResource]));

                // Get items
                const permissionsIn = await this.services.permissionService.list(getParams(dataExplorer, groupUuid));
                api.dispatch(updateResources(permissionsIn.items));

                api.dispatch(GroupMembersPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(permissionsIn),
                    items: permissionsIn.items.map(resource => resource.uuid),
                }));

                const userUuids = permissionsIn.items
                    .filter((item) => item.tailKind === ResourceKind.USER)
                    .map(item => item.tailUuid);
                if (userUuids.length) {
                    this.services.userService
                        .list(getTypeParams(dataExplorer, userUuids))
                        .then((usersIn: ListResults<UserResource>) => (
                            api.dispatch(updateResources(usersIn.items))
                        ));
                }

                const projectUuids = permissionsIn.items
                    .filter((item) => item.tailKind === ResourceKind.PROJECT)
                    .map(item => item.tailUuid);
                if (projectUuids.length) {
                    this.services.projectService
                        .list(getTypeParams(dataExplorer, projectUuids))
                        .then((projectsIn: ListResults<ProjectResource>) => (
                            api.dispatch(updateResources(projectsIn.items))
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
                        api.dispatch<any>(GroupMembersPanelActions.SET_ITEMS_AVAILABLE(results.itemsAvailable));
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

export const getTypeParams = (dataExplorer: DataExplorer, uuids: string[]): ListArguments => ({
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
        .addEqual('head_uuid', groupUuid)
        .addEqual('link_class', LinkClass.PERMISSION)
        .getFilters();
};

const couldNotFetchGroupDetailsContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch group members.',
        kind: SnackbarKind.ERROR
    });
