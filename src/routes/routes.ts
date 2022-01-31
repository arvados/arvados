// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { matchPath } from 'react-router';
import { ResourceKind, RESOURCE_UUID_PATTERN, extractUuidKind, COLLECTION_PDH_REGEX, PORTABLE_DATA_HASH_PATTERN } from 'models/resource';
import { getProjectUrl } from 'models/project';
import { getCollectionUrl } from 'models/collection';
import { Config } from 'common/config';
import { Session } from "models/session";

export interface FederationConfig {
    localCluster: string;
    remoteHostsConfig: { [key: string]: Config };
    sessions: Session[];
}

export const Routes = {
    ROOT: '/',
    TOKEN: '/token',
    FED_LOGIN: '/fedtoken',
    ADD_SESSION: '/add-session',
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
    LINK_ACCOUNT: '/link_account',
    KEEP_SERVICES: `/keep-services`,
    USERS: '/users',
    API_CLIENT_AUTHORIZATIONS: `/api_client_authorizations`,
    GROUPS: '/groups',
    GROUP_DETAILS: `/group/:id(${RESOURCE_UUID_PATTERN})`,
    LINKS: '/links',
    PUBLIC_FAVORITES: '/public-favorites',
    COLLECTIONS_CONTENT_ADDRESS: `/collections/:id(${PORTABLE_DATA_HASH_PATTERN})`,
    ALL_PROCESSES: '/all_processes',
    NO_MATCH: '*',
};

export const getResourceUrl = (uuid: string) => {
    const kind = extractUuidKind(uuid);
    switch (kind) {
        case ResourceKind.PROJECT:
            return getProjectUrl(uuid);
        case ResourceKind.USER:
            return getProjectUrl(uuid);
        case ResourceKind.COLLECTION:
            return getCollectionUrl(uuid);
        case ResourceKind.PROCESS:
            return getProcessUrl(uuid);
        default:
            return undefined;
    }
};

export const getNavUrl = (uuid: string, config: FederationConfig) => {
    const path = getResourceUrl(uuid) || "";
    const cls = uuid.substring(0, 5);
    if (cls === config.localCluster || extractUuidKind(uuid) === ResourceKind.USER || COLLECTION_PDH_REGEX.exec(uuid)) {
        return path;
    } else if (config.remoteHostsConfig[cls]) {
        let u: URL;
        if (config.remoteHostsConfig[cls].workbench2Url) {
	    /* NOTE: wb2 presently doesn't support passing api_token
	       to arbitrary page to set credentials, only through
	       api-token route.  So for navigation to work, user needs
	       to already be logged in.  In the future we want to just
	       request the records and display in the current
	       workbench instance making this redirect unnecessary. */
            u = new URL(config.remoteHostsConfig[cls].workbench2Url);
        } else {
            u = new URL(config.remoteHostsConfig[cls].workbenchUrl);
            u.search = "api_token=" + config.sessions.filter((s) => s.clusterId === cls)[0].token;
        }
        u.pathname = path;
        return u.toString();
    } else {
        return "";
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

export const matchAllProcessesRoute = (route: string) =>
    matchPath(route, { path: Routes.ALL_PROCESSES });

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

export const matchLinkAccountRoute = (route: string) =>
    matchPath(route, { path: Routes.LINK_ACCOUNT });

export const matchKeepServicesRoute = (route: string) =>
    matchPath(route, { path: Routes.KEEP_SERVICES });

export const matchTokenRoute = (route: string) =>
    matchPath(route, { path: Routes.TOKEN });

export const matchFedTokenRoute = (route: string) =>
    matchPath(route, { path: Routes.FED_LOGIN });

export const matchUsersRoute = (route: string) =>
    matchPath(route, { path: Routes.USERS });

export const matchApiClientAuthorizationsRoute = (route: string) =>
    matchPath(route, { path: Routes.API_CLIENT_AUTHORIZATIONS });

export const matchGroupsRoute = (route: string) =>
    matchPath(route, { path: Routes.GROUPS });

export const matchGroupDetailsRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.GROUP_DETAILS });

export const matchLinksRoute = (route: string) =>
    matchPath(route, { path: Routes.LINKS });

export const matchPublicFavoritesRoute = (route: string) =>
    matchPath(route, { path: Routes.PUBLIC_FAVORITES });

export const matchCollectionsContentAddressRoute = (route: string) =>
    matchPath(route, { path: Routes.COLLECTIONS_CONTENT_ADDRESS });
