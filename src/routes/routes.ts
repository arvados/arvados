// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { RootStore } from '~/store/store';
import { matchPath } from 'react-router';
import { ResourceKind, RESOURCE_UUID_PATTERN, extractUuidKind } from '~/models/resource';
import { getProjectUrl } from '../models/project';
import { getCollectionUrl } from '~/models/collection';
import { loadProject, loadFavorites, loadCollection, loadProcessLog } from '~/store/workbench/workbench-actions';
import { loadProcess } from '~/store/processes/processes-actions';

export const Routes = {
    ROOT: '/',
    TOKEN: '/token',
    PROJECTS: `/projects/:id(${RESOURCE_UUID_PATTERN})`,
    COLLECTIONS: `/collections/:id(${RESOURCE_UUID_PATTERN})`,
    PROCESSES: `/processes/:id(${RESOURCE_UUID_PATTERN})`,
    FAVORITES: '/favorites',
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

export const addRouteChangeHandlers = (history: History, store: RootStore) => {
    const handler = handleLocationChange(store);
    handler(history.location);
    history.listen(handler);
};

export const matchRootRoute = (route: string) =>
    matchPath(route, { path: Routes.ROOT, exact: true });

export const matchFavoritesRoute = (route: string) =>
    matchPath(route, { path: Routes.FAVORITES });

export interface ResourceRouteParams {
    id: string;
}

export const matchProjectRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROJECTS });

export const matchCollectionRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.COLLECTIONS });

export const matchProcessRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROCESSES });

export const matchProcessLogRoute = (route: string) =>
    matchPath<ResourceRouteParams>(route, { path: Routes.PROCESS_LOGS });

const handleLocationChange = (store: RootStore) => ({ pathname }: Location) => {
    const projectMatch = matchProjectRoute(pathname);
    const collectionMatch = matchCollectionRoute(pathname);
    const favoriteMatch = matchFavoritesRoute(pathname);
    const processMatch = matchProcessRoute(pathname);
    const processLogMatch = matchProcessLogRoute(pathname);
    
    if (projectMatch) {
        store.dispatch(loadProject(projectMatch.params.id));
    } else if (collectionMatch) {
        store.dispatch(loadCollection(collectionMatch.params.id));
    } else if (favoriteMatch) {
        store.dispatch(loadFavorites());
    } else if (processMatch) {
        store.dispatch(loadProcess(processMatch.params.id));
    } else if (processLogMatch) {
        store.dispatch(loadProcessLog(processLogMatch.params.id));
    }
};
