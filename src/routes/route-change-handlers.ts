// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { RootStore } from 'store/store';
import * as Routes from 'routes/routes';
import * as WorkbenchActions from 'store/workbench/workbench-actions';
import { navigateToRootProject } from 'store/navigation/navigation-action';
import { dialogActions } from 'store/dialog/dialog-actions';
import { contextMenuActions } from 'store/context-menu/context-menu-actions';
import { searchBarActions } from 'store/search-bar/search-bar-actions';
import { pluginConfig } from 'plugins';

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
    const publicFavoritesMatch = Routes.matchPublicFavoritesRoute(pathname);
    const trashMatch = Routes.matchTrashRoute(pathname);
    const processMatch = Routes.matchProcessRoute(pathname);
    const processLogMatch = Routes.matchProcessLogRoute(pathname);
    const repositoryMatch = Routes.matchRepositoriesRoute(pathname);
    const searchResultsMatch = Routes.matchSearchResultsRoute(pathname);
    const sharedWithMeMatch = Routes.matchSharedWithMeRoute(pathname);
    const runProcessMatch = Routes.matchRunProcessRoute(pathname);
    const virtualMachineUserMatch = Routes.matchUserVirtualMachineRoute(pathname);
    const virtualMachineAdminMatch = Routes.matchAdminVirtualMachineRoute(pathname);
    const workflowMatch = Routes.matchWorkflowRoute(pathname);
    const sshKeysUserMatch = Routes.matchSshKeysUserRoute(pathname);
    const sshKeysAdminMatch = Routes.matchSshKeysAdminRoute(pathname);
    const siteManagerMatch = Routes.matchSiteManagerRoute(pathname);
    const keepServicesMatch = Routes.matchKeepServicesRoute(pathname);
    const apiClientAuthorizationsMatch = Routes.matchApiClientAuthorizationsRoute(pathname);
    const myAccountMatch = Routes.matchMyAccountRoute(pathname);
    const linkAccountMatch = Routes.matchLinkAccountRoute(pathname);
    const userMatch = Routes.matchUsersRoute(pathname);
    const groupsMatch = Routes.matchGroupsRoute(pathname);
    const groupDetailsMatch = Routes.matchGroupDetailsRoute(pathname);
    const linksMatch = Routes.matchLinksRoute(pathname);
    const collectionsContentAddressMatch = Routes.matchCollectionsContentAddressRoute(pathname);
    const allProcessesMatch = Routes.matchAllProcessesRoute(pathname);

    store.dispatch(dialogActions.CLOSE_ALL_DIALOGS());
    store.dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
    store.dispatch(searchBarActions.CLOSE_SEARCH_VIEW());

    for (const locChangeFn of pluginConfig.locationChangeHandlers) {
        if (locChangeFn(store, pathname)) {
            return;
        }
    }

    if (projectMatch) {
        store.dispatch(WorkbenchActions.loadProject(projectMatch.params.id));
    } else if (collectionMatch) {
        store.dispatch(WorkbenchActions.loadCollection(collectionMatch.params.id));
    } else if (favoriteMatch) {
        store.dispatch(WorkbenchActions.loadFavorites());
    } else if (publicFavoritesMatch) {
        store.dispatch(WorkbenchActions.loadPublicFavorites());
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
    } else if (virtualMachineUserMatch) {
        store.dispatch(WorkbenchActions.loadVirtualMachines);
    } else if (virtualMachineAdminMatch) {
        store.dispatch(WorkbenchActions.loadVirtualMachines);
    } else if (repositoryMatch) {
        store.dispatch(WorkbenchActions.loadRepositories);
    } else if (sshKeysUserMatch) {
        store.dispatch(WorkbenchActions.loadSshKeys);
    } else if (sshKeysAdminMatch) {
        store.dispatch(WorkbenchActions.loadSshKeys);
    } else if (siteManagerMatch) {
        store.dispatch(WorkbenchActions.loadSiteManager);
    } else if (keepServicesMatch) {
        store.dispatch(WorkbenchActions.loadKeepServices);
    } else if (apiClientAuthorizationsMatch) {
        store.dispatch(WorkbenchActions.loadApiClientAuthorizations);
    } else if (myAccountMatch) {
        store.dispatch(WorkbenchActions.loadMyAccount);
    } else if (linkAccountMatch) {
        store.dispatch(WorkbenchActions.loadLinkAccount);
    } else if (userMatch) {
        store.dispatch(WorkbenchActions.loadUsers);
    } else if (groupsMatch) {
        store.dispatch(WorkbenchActions.loadGroupsPanel);
    } else if (groupDetailsMatch) {
        store.dispatch(WorkbenchActions.loadGroupDetailsPanel(groupDetailsMatch.params.id));
    } else if (linksMatch) {
        store.dispatch(WorkbenchActions.loadLinks);
    } else if (collectionsContentAddressMatch) {
        store.dispatch(WorkbenchActions.loadCollectionContentAddress);
    } else if (allProcessesMatch) {
        store.dispatch(WorkbenchActions.loadAllProcesses());
    }
};
