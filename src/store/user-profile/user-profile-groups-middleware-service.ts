// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ServiceRepository } from 'services/services';
import { MiddlewareAPI, Dispatch } from 'redux';
import { DataExplorerMiddlewareService, listResultsToDataExplorerItemsMeta } from 'store/data-explorer/data-explorer-middleware-service';
import { RootState } from 'store/store';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";
import { getCurrentUserProfilePanelUuid, UserProfileGroupsActions } from 'store/user-profile/user-profile-actions';
import { updateResources } from 'store/resources/resources-actions';
import { FilterBuilder } from 'services/api/filter-builder';
import { LinkClass } from 'models/link';
import { ResourceKind } from 'models/resource';
import { GroupClass } from 'models/group';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

export class UserProfileGroupsMiddlewareService extends DataExplorerMiddlewareService {
    constructor(private services: ServiceRepository, id: string) {
        super(id);
    }

    async requestItems(api: MiddlewareAPI<Dispatch, RootState>) {
        const state = api.getState();
        const userUuid = getCurrentUserProfilePanelUuid(state.properties);
        try {
            api.dispatch(progressIndicatorActions.START_WORKING(this.getId()));

            // Get user
            const user = await this.services.userService.get(userUuid || '');
            api.dispatch(updateResources([user]));

            // Get user's group memberships
            const groupMembershipLinks = await this.services.permissionService.list({
                filters: new FilterBuilder()
                    .addEqual('tail_uuid', userUuid)
                    .addEqual('link_class', LinkClass.PERMISSION)
                    .addEqual('head_kind', ResourceKind.GROUP)
                    .getFilters()
            });
            // Update resources, includes "project" groups
            api.dispatch(updateResources(groupMembershipLinks.items));

            // Get user's groups details and filter to role groups
            const groups = await this.services.groupsService.list({
                filters: new FilterBuilder()
                    .addIn('uuid', groupMembershipLinks.items
                        .map(item => item.headUuid))
                    .addEqual('group_class', GroupClass.ROLE)
                    .getFilters(),
                count: "none"
            });
            api.dispatch(updateResources(groups.items));

            // Get permission links for only role groups
            const roleGroupMembershipLinks = await this.services.permissionService.list({
                filters: new FilterBuilder()
                    .addIn('head_uuid', groups.items.map(item => item.uuid))
                    .addEqual('tail_uuid', userUuid)
                    .addEqual('link_class', LinkClass.PERMISSION)
                    .addEqual('head_kind', ResourceKind.GROUP)
                    .getFilters()
            });

            api.dispatch(UserProfileGroupsActions.SET_ITEMS({
                ...listResultsToDataExplorerItemsMeta(roleGroupMembershipLinks),
                items: roleGroupMembershipLinks.items.map(item => item.uuid),
            }));
        } catch {
            api.dispatch(couldNotFetchGroups());
        } finally {
            api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
        }
    }
}

const couldNotFetchGroups = () =>
    snackbarActions.OPEN_SNACKBAR({
        message: 'Could not fetch groups.',
        kind: SnackbarKind.ERROR
    });
