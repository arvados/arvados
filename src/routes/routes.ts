// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { matchPath } from 'react-router';
import { ResourceKind, RESOURCE_UUID_PATTERN, extractUuidKind } from '~/models/resource';
import { getProjectUrl } from '~/models/project';
import { getCollectionUrl } from '~/models/collection';

export const Routes = {
    ROOT: '/',
    TOKEN: '/token',
    PROJECTS: `/projects/:id(${RESOURCE_UUID_PATTERN})`,
    COLLECTIONS: `/collections/:id(${RESOURCE_UUID_PATTERN})`,
    PROCESSES: `/processes/:id(${RESOURCE_UUID_PATTERN})`,
    FAVORITES: '/favorites',
    TRASH: '/trash',
    PROCESS_LOGS: `/process-logs/:id(${RESOURCE_UUID_PATTERN})`,
    REPOSITORIES: '/repositories',
    SHARED_WITH_ME: '/shared-with-me',
    RUN_PROCESS: '/run-process',
    VIRTUAL_MACHINES_ADMIN: '/virtual-machines-admin',
    VIRTUAL_MACHINES_USER: '/virtual-machines-user',
    WORKFLOWS: '/workflows',
    SEARCH_RESULTS: '/search-results',
    SSH_KEYS_ADMIN: `/ssh-keys-admin`,
    SSH_KEYS_USER: `/ssh-keys-user`,
    SITE_MANAGER: `/site-manager`,
    MY_ACCOUNT: '/my-account',
    KEEP_SERVICES: `/keep-services`,
    COMPUTE_NODES: `/nodes`,
    USERS: '/users',
    API_CLIENT_AUTHORIZATIONS: `/api_client_authorizations`,
    GROUPS: '/groups',
    GROUP_DETAILS: `/group/:id(${RESOURCE_UUID_PATTERN})`,
    LINKS: '/links'
};

export const getResourceUrl = (uuid: string) => {
    const kind = extractUuidKind(uuid);
    switch (kind) {
        case ResourceKind.PROJECT:
            return getProjectUrl(uuid);
        case ResourceKind.COLLECTION:
            return getCollectionUrl(uuid);
        case ResourceKind.PROCESS:
            return getProcessUrl(uuid);
        default:
            return undefined;
    }
};

export const getProcessUrl = (uuid: string) => `/processes/${uuid}`;

export const getProcessLogUrl = (uuid: string) => `/process-logs/${uuid}`;

export const getGroupUrl = (uuid: string) => `/group/${uuid}`;

export interface ResourceRouteParams {
    id: string;
}

export const matchRootRoute = (route: string) =>
    matchPath(route, { path: Routes.ROOT, exact: true });

export const matchFavoritesRoute = (route: string) =>
    matchPath(route, { path: Routes.FAVORITES });

export const matchTrashRoute = (route: string) =>
    matchPath(route, { path: Routes.TRASH });

export const matchProjectRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROJECTS });

export const matchCollectionRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.COLLECTIONS });

export const matchProcessRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROCESSES });

export const matchProcessLogRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROCESS_LOGS });

export const matchSharedWithMeRoute = (route: string) =>
    matchPath(route, { path: Routes.SHARED_WITH_ME });

export const matchRunProcessRoute = (route: string) =>
    matchPath(route, { path: Routes.RUN_PROCESS });

export const matchWorkflowRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.WORKFLOWS });

export const matchSearchResultsRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.SEARCH_RESULTS });

export const matchUserVirtualMachineRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.VIRTUAL_MACHINES_USER });

export const matchAdminVirtualMachineRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.VIRTUAL_MACHINES_ADMIN });

export const matchRepositoriesRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.REPOSITORIES });

export const matchSshKeysUserRoute = (route: string) =>
    matchPath(route, { path: Routes.SSH_KEYS_USER });

export const matchSshKeysAdminRoute = (route: string) =>
    matchPath(route, { path: Routes.SSH_KEYS_ADMIN });

export const matchSiteManagerRoute = (route: string) =>
    matchPath(route, { path: Routes.SITE_MANAGER });

export const matchMyAccountRoute = (route: string) =>
    matchPath(route, { path: Routes.MY_ACCOUNT });

export const matchKeepServicesRoute = (route: string) =>
    matchPath(route, { path: Routes.KEEP_SERVICES });

export const matchUsersRoute = (route: string) =>
    matchPath(route, { path: Routes.USERS });

export const matchComputeNodesRoute = (route: string) =>
    matchPath(route, { path: Routes.COMPUTE_NODES });

export const matchApiClientAuthorizationsRoute = (route: string) =>
    matchPath(route, { path: Routes.API_CLIENT_AUTHORIZATIONS });

export const matchGroupsRoute = (route: string) =>
    matchPath(route, { path: Routes.GROUPS });

export const matchGroupDetailsRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.GROUP_DETAILS });
    
export const matchLinksRoute = (route: string) =>
    matchPath(route, { path: Routes.LINKS });
