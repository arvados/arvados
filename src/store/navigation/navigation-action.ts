// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { push } from "react-router-redux";
import { ResourceKind, extractUuidKind } from '~/models/resource';
import { getCollectionUrl } from "~/models/collection";
import { getProjectUrl } from "~/models/project";
import { SidePanelTreeCategory } from '../side-panel-tree/side-panel-tree-actions';
import { Routes, getProcessUrl, getProcessLogUrl, getGroupUrl } from '~/routes/routes';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { GROUPS_PANEL_LABEL } from '~/store/breadcrumbs/breadcrumbs-actions';

export const navigateTo = (uuid: string) =>
    async (dispatch: Dispatch) => {
        const kind = extractUuidKind(uuid);
        if (kind === ResourceKind.PROJECT || kind === ResourceKind.USER) {
            dispatch<any>(navigateToProject(uuid));
        } else if (kind === ResourceKind.COLLECTION) {
            dispatch<any>(navigateToCollection(uuid));
        } else if (kind === ResourceKind.CONTAINER_REQUEST) {
            dispatch<any>(navigateToProcess(uuid));
        } else if (kind === ResourceKind.VIRTUAL_MACHINE) {
            dispatch<any>(navigateToAdminVirtualMachines);
        }
        if (uuid === SidePanelTreeCategory.FAVORITES) {
            dispatch<any>(navigateToFavorites);
        } else if (uuid === SidePanelTreeCategory.SHARED_WITH_ME) {
            dispatch(navigateToSharedWithMe);
        } else if (uuid === SidePanelTreeCategory.WORKFLOWS) {
            dispatch(navigateToWorkflows);
        } else if (uuid === SidePanelTreeCategory.TRASH) {
            dispatch(navigateToTrash);
        } else if (uuid === GROUPS_PANEL_LABEL) {
            dispatch(navigateToGroups);
        }
    };

export const navigateToRoot = push(Routes.ROOT);

export const navigateToFavorites = push(Routes.FAVORITES);

export const navigateToTrash = push(Routes.TRASH);

export const navigateToWorkflows = push(Routes.WORKFLOWS);

export const navigateToProject = compose(push, getProjectUrl);

export const navigateToCollection = compose(push, getCollectionUrl);

export const navigateToProcess = compose(push, getProcessUrl);

export const navigateToProcessLogs = compose(push, getProcessLogUrl);

export const navigateToRootProject = (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const rootProjectUuid = services.authService.getUuid();
    if (rootProjectUuid) {
        dispatch(navigateToProject(rootProjectUuid));
    }
};

export const navigateToSharedWithMe = push(Routes.SHARED_WITH_ME);

export const navigateToRunProcess = push(Routes.RUN_PROCESS);

export const navigateToSearchResults = push(Routes.SEARCH_RESULTS);

export const navigateToUserVirtualMachines = push(Routes.VIRTUAL_MACHINES_USER);

export const navigateToAdminVirtualMachines = push(Routes.VIRTUAL_MACHINES_ADMIN);

export const navigateToRepositories = push(Routes.REPOSITORIES);

export const navigateToSshKeysAdmin= push(Routes.SSH_KEYS_ADMIN);

export const navigateToSshKeysUser= push(Routes.SSH_KEYS_USER);

export const navigateToSiteManager= push(Routes.SITE_MANAGER);

export const navigateToMyAccount = push(Routes.MY_ACCOUNT);

export const navigateToKeepServices = push(Routes.KEEP_SERVICES);

export const navigateToComputeNodes = push(Routes.COMPUTE_NODES);

export const navigateToUsers = push(Routes.USERS);

export const navigateToApiClientAuthorizations = push(Routes.API_CLIENT_AUTHORIZATIONS);

export const navigateToGroups = push(Routes.GROUPS);

export const navigateToGroupDetails = compose(push, getGroupUrl);

export const navigateToLinks = push(Routes.LINKS);
