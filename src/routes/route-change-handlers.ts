// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { RootStore } from '~/store/store';
import { matchProcessRoute, matchProcessLogRoute, matchProjectRoute, matchCollectionRoute, matchFavoritesRoute, matchTrashRoute, matchRootRoute } from './routes';
import { loadProject, loadCollection, loadFavorites, loadTrash, loadProcess, loadProcessLog } from '~/store/workbench/workbench-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';

export const addRouteChangeHandlers = (history: History, store: RootStore) => {
    const handler = handleLocationChange(store);
    handler(history.location);
    history.listen(handler);
};

const handleLocationChange = (store: RootStore) => ({ pathname }: Location) => {
    const rootMatch = matchRootRoute(pathname);
    const projectMatch = matchProjectRoute(pathname);
    const collectionMatch = matchCollectionRoute(pathname);
    const favoriteMatch = matchFavoritesRoute(pathname);
    const trashMatch = matchTrashRoute(pathname);
    const processMatch = matchProcessRoute(pathname);
    const processLogMatch = matchProcessLogRoute(pathname);
    
    if (projectMatch) {
        store.dispatch(loadProject(projectMatch.params.id));
    } else if (collectionMatch) {
        store.dispatch(loadCollection(collectionMatch.params.id));
    } else if (favoriteMatch) {
        store.dispatch(loadFavorites());
    } else if (trashMatch) {
        store.dispatch(loadTrash());
    } else if (processMatch) {
        store.dispatch(loadProcess(processMatch.params.id));
    } else if (processLogMatch) {
        store.dispatch(loadProcessLog(processLogMatch.params.id));
    } else if (rootMatch) {
        store.dispatch(navigateToRootProject);
    }
};
