// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { RootStore } from '~/store/store';
import * as Routes from '~/routes/routes';
import * as WorkbenchActions from '~/store/workbench/workbench-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';

export const addRouteChangeHandlers = (history: History, store: RootStore) => {
    const handler = handleLocationChange(store);
    handler(history.location);
    history.listen(handler);
};

const handleLocationChange = (store: RootStore) => ({ pathname }: Location) => {
    const rootMatch = Routes.matchRootRoute(pathname);
    const projectMatch = Routes.matchProjectRoute(pathname);
    const collectionMatch = Routes.matchCollectionRoute(pathname);
    const favoriteMatch = Routes.matchFavoritesRoute(pathname);
    const trashMatch = Routes.matchTrashRoute(pathname);
    const processMatch = Routes.matchProcessRoute(pathname);
    const processLogMatch = Routes.matchProcessLogRoute(pathname);
    const repositoryMatch = Routes.matchRepositoriesRoute(pathname);
    const searchResultsMatch = Routes.matchSearchResultsRoute(pathname);
    const sharedWithMeMatch = Routes.matchSharedWithMeRoute(pathname);
    const runProcessMatch = Routes.matchRunProcessRoute(pathname);
    const virtualMachineMatch = Routes.matchVirtualMachineRoute(pathname);
    const workflowMatch = Routes.matchWorkflowRoute(pathname);
    const sshKeysMatch = Routes.matchSshKeysRoute(pathname);
    const keepServicesMatch = Routes.matchKeepServicesRoute(pathname);
    const computeNodesMatch = Routes.matchComputeNodesRoute(pathname);
    const apiClientAuthorizationsMatch = Routes.matchApiClientAuthorizationsRoute(pathname);
    const myAccountMatch = Routes.matchMyAccountRoute(pathname);
    const userMatch = Routes.matchUsersRoute(pathname);

    if (projectMatch) {
        store.dispatch(WorkbenchActions.loadProject(projectMatch.params.id));
    } else if (collectionMatch) {
        store.dispatch(WorkbenchActions.loadCollection(collectionMatch.params.id));
    } else if (favoriteMatch) {
        store.dispatch(WorkbenchActions.loadFavorites());
    } else if (trashMatch) {
        store.dispatch(WorkbenchActions.loadTrash());
    } else if (processMatch) {
        store.dispatch(WorkbenchActions.loadProcess(processMatch.params.id));
    } else if (processLogMatch) {
        store.dispatch(WorkbenchActions.loadProcessLog(processLogMatch.params.id));
    } else if (rootMatch) {
        store.dispatch(navigateToRootProject);
    } else if (sharedWithMeMatch) {
        store.dispatch(WorkbenchActions.loadSharedWithMe);
    } else if (runProcessMatch) {
        store.dispatch(WorkbenchActions.loadRunProcess);
    } else if (workflowMatch) {
        store.dispatch(WorkbenchActions.loadWorkflow);
    } else if (searchResultsMatch) {
        store.dispatch(WorkbenchActions.loadSearchResults);
    } else if (virtualMachineMatch) {
        store.dispatch(WorkbenchActions.loadVirtualMachines);
    } else if(repositoryMatch) {
        store.dispatch(WorkbenchActions.loadRepositories);
    } else if (sshKeysMatch) {
        store.dispatch(WorkbenchActions.loadSshKeys);
    } else if (keepServicesMatch) {
        store.dispatch(WorkbenchActions.loadKeepServices);
    } else if (computeNodesMatch) {
        store.dispatch(WorkbenchActions.loadComputeNodes);
    } else if (apiClientAuthorizationsMatch) {
        store.dispatch(WorkbenchActions.loadApiClientAuthorizations);
    } else if (myAccountMatch) {
        store.dispatch(WorkbenchActions.loadMyAccount);
    }else if (userMatch) {
        store.dispatch(WorkbenchActions.loadUsers);
    }
};
