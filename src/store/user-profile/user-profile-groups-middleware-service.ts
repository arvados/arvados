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
            api.dispatch(updateResources(groupMembershipLinks.items));

            // Get user's groups details
            const groups = await this.services.groupsService.list({
                filters: new FilterBuilder()
                    .addIn('uuid', groupMembershipLinks.items
                        .map(item => item.headUuid))
                    .getFilters(),
                count: "none"
            });
            api.dispatch(updateResources(groups.items));

            api.dispatch(UserProfileGroupsActions.SET_ITEMS({
                ...listResultsToDataExplorerItemsMeta(groupMembershipLinks),
                items: groupMembershipLinks.items.map(item => item.uuid),
            }));
        } catch {
            // api.dispatch(couldNotFetchUsers());
        } finally {
            api.dispatch(progressIndicatorActions.STOP_WORKING(this.getId()));
        }
    }
}
