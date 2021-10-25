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
import { getCurrentGroupDetailsPanelUuid, GroupPermissionsPanelActions } from 'store/group-details-panel/group-details-panel-actions';
import { LinkClass } from 'models/link';
import { ResourceKind } from 'models/resource';

export class GroupDetailsPanelPermissionsMiddlewareService extends DataExplorerMiddlewareService {

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
                const permissionsOut = await this.services.permissionService.list({
                    filters: new FilterBuilder()
                        .addEqual('tail_uuid', groupUuid)
                        .addEqual('link_class', LinkClass.PERMISSION)
                        .getFilters()
                });
                api.dispatch(updateResources(permissionsOut.items));

                api.dispatch(GroupPermissionsPanelActions.SET_ITEMS({
                    ...listResultsToDataExplorerItemsMeta(permissionsOut),
                    items: permissionsOut.items.map(item => item.uuid),
                }));

                const usersOut = await this.services.userService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', permissionsOut.items
                            .filter((item) => item.headKind === ResourceKind.USER)
                            .map(item => item.headUuid))
                        .getFilters(),
                    count: "none"
                });
                api.dispatch(updateResources(usersOut.items));

                const collectionsOut = await this.services.collectionService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', permissionsOut.items
                            .filter((item) => item.headKind === ResourceKind.COLLECTION)
                            .map(item => item.headUuid))
                        .getFilters(),
                    count: "none"
                });
                api.dispatch(updateResources(collectionsOut.items));

                const projectsOut = await this.services.projectService.list({
                    filters: new FilterBuilder()
                        .addIn('uuid', permissionsOut.items
                            .filter((item) => item.headKind === ResourceKind.PROJECT)
                            .map(item => item.headUuid))
                        .getFilters(),
                    count: "none"
                });
                api.dispatch(updateResources(projectsOut.items));
            } catch (e) {
                api.dispatch(couldNotFetchGroupDetailsContents());
            }
        }
    }
}

const groupsDetailsPanelDataExplorerIsNotSet = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Group permissions panel is not ready.',
        kind: SnackbarKind.ERROR
    });

const couldNotFetchGroupDetailsContents = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch group permissions.',
        kind: SnackbarKind.ERROR
    });
