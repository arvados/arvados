// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { RootStore } from '~/store/store';
import { matchProcessRoute, matchProcessLogRoute, matchProjectRoute, matchCollectionRoute, matchFavoritesRoute, matchTrashRoute, matchRootRoute, matchSharedWithMeRoute, matchRunProcessRoute, matchWorkflowRoute, matchSearchResultsRoute, matchSshKeysRoute, matchRepositoriesRoute } from './routes';
import { loadProject, loadCollection, loadFavorites, loadTrash, loadProcess, loadProcessLog, loadSshKeys, loadRepositories } from '~/store/workbench/workbench-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';
import { loadSharedWithMe, loadRunProcess, loadWorkflow, loadSearchResults } from '~//store/workbench/workbench-actions';

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
    const repositoryMatch = matchRepositoriesRoute(pathname); 
    const searchResultsMatch = matchSearchResultsRoute(pathname);
    const sharedWithMeMatch = matchSharedWithMeRoute(pathname);
    const runProcessMatch = matchRunProcessRoute(pathname);
    const workflowMatch = matchWorkflowRoute(pathname);
    const sshKeysMatch = matchSshKeysRoute(pathname);

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
    } else if (sharedWithMeMatch) {
        store.dispatch(loadSharedWithMe);
    } else if (runProcessMatch) {
        store.dispatch(loadRunProcess);
    } else if (workflowMatch) {
        store.dispatch(loadWorkflow);
    } else if (searchResultsMatch) {
        store.dispatch(loadSearchResults);
    } else if(repositoryMatch) {
        store.dispatch(loadRepositories);
    } else if (sshKeysMatch) {
        store.dispatch(loadSshKeys);
    }
};
