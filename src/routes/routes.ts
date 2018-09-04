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
    PROCESS_LOGS: `/process-logs/:id(${RESOURCE_UUID_PATTERN})`
};

export const getResourceUrl = (uuid: string) => {
    const kind = extractUuidKind(uuid);
    switch (kind) {
        case ResourceKind.PROJECT:
            return getProjectUrl(uuid);
        case ResourceKind.COLLECTION:
            return getCollectionUrl(uuid);
        default:
            return undefined;
    }
};

export const getProcessUrl = (uuid: string) => `/processes/${uuid}`;

export const getProcessLogUrl = (uuid: string) => `/process-logs/${uuid}`;

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
