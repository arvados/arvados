// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, MiddlewareAPI } from "redux";
import { DataExplorerMiddlewareService, listResultsToDataExplorerItemsMeta } from "store/data-explorer/data-explorer-middleware-service";
import { RootState } from "store/store";
import { ServiceRepository } from "services/services";
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getDataExplorer } from "store/data-explorer/data-explorer-reducer";
import { FilterBuilder } from 'services/api/filter-builder';
import { updateResources } from 'store/resources/resources-actions';
import { getCurrentGroupDetailsPanelUuid, GroupMembersPanelActions } from 'store/group-details-panel/group-details-panel-actions';
import { LinkClass } from 'models/link';
import { ResourceKind } from 'models/resource';

export class GroupDetailsPanelMembersMiddlewareService extends DataExplorerMiddlewareService {

    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const dataExplorer = getDataExplorer(api.getState().dataExplorer, this.getId());
        const groupUuid = getCurrentGroupDetailsPanelUuid(api.getState().properties);
        if (!dataExplorer || !groupUuid) {
            api.dispatch(groupsDetailsPanelDataExplorerIsNotSet());
        } else {
            try {
                const groupResource = await this.services.groupsService.get(groupUuid);
                api.dispatch(updateResources([groupResource]));

                const permissionsIn = await this.services.permissionService.list({
                    filters: new FilterBuilder()
                        .addEqual('head_uuid', groupUuid)
                        .addEqual('link_class', LinkClass.PERMISSION)
                        .getFilters()
                });
                api.dispatch(updateResources(permissionsIn.items));

                api.dispatch(GroupMembersPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(permissionsIn),
                    items: permissionsIn.items.map(item => item.uuid),
                }));

                const usersIn = await this.services.userService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', permissionsIn.items
                            .filter((item) => item.tailKind === ResourceKind.USER)
                            .map(item => item.tailUuid))
                        .getFilters(),
                    count: "none"
                });
                api.dispatch(updateResources(usersIn.items));

                const projectsIn = await this.services.projectService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', permissionsIn.items
                            .filter((item) => item.tailKind === ResourceKind.PROJECT)
                            .map(item => item.tailUuid))
                        .getFilters(),
                    count: "none"
                });
                api.dispatch(updateResources(projectsIn.items));
            } catch (e) {
                api.dispatch(couldNotFetchGroupDetailsContents());
            }
        }
    }
}

const groupsDetailsPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Group members panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchGroupDetailsContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch group members.',
        kind: SnackbarKind.ERROR
    });
