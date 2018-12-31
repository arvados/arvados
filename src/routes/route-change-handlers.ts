// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { History, Location } from 'history';
import { match } from 'react-router-dom';
import { Dispatch } from 'redux';
import { ThunkAction } from 'redux-thunk';
import { RootStore } from '~/store/store';
import * as R from '~/routes/routes';
import * as WA from '~/store/workbench/workbench-actions';
import { navigateToRootProject } from '~/store/navigation/navigation-action';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { contextMenuActions } from '~/store/context-menu/context-menu-actions';
import { searchBarActions } from '~/store/search-bar/search-bar-actions';

export const addRouteChangeHandlers = (history: History, store: RootStore) => {
    const handler = handleLocationChange(store);
    handler(history.location);
    history.listen(handler);
};

const handleLocationChange = (store: RootStore) => ({ pathname }: Location) => {

    store.dispatch(dialogActions.CLOSE_ALL_DIALOGS());
    store.dispatch(contextMenuActions.CLOSE_CONTEXT_MENU());
    store.dispatch(searchBarActions.CLOSE_SEARCH_VIEW());

    locationChangeHandlers.find(handler => handler(store.dispatch, pathname));

};

type MatchRoute<Params> = (route: string) => match<Params> | null;
type ActionCreator<Params> = (params: Params) => ThunkAction<any, any, any, any>;

const handle = <Params>(matchRoute: MatchRoute<Params>, actionCreator: ActionCreator<Params>) =>
    (dispatch: Dispatch, route: string) => {
        const match = matchRoute(route);
        return match
            ? (
                dispatch<any>(actionCreator(match.params)),
                true
            )
            : false;
    };

const locationChangeHandlers = [

    handle(
        R.matchApiClientAuthorizationsRoute,
        () => WA.loadApiClientAuthorizations
    ),

    handle(
        R.matchCollectionRoute,
        ({ id }) => WA.loadCollection(id)
    ),

    handle(
        R.matchComputeNodesRoute,
        () => WA.loadComputeNodes
    ),

    handle(
        R.matchFavoritesRoute,
        () => WA.loadFavorites
    ),

    handle(
        R.matchGroupDetailsRoute,
        ({ id }) => WA.loadGroupDetailsPanel(id)
    ),

    handle(
        R.matchGroupsRoute,
        () => WA.loadGroupsPanel
    ),

    handle(
        R.matchKeepServicesRoute,
        () => WA.loadKeepServices
    ),

    handle(
        R.matchLinksRoute,
        () => WA.loadLinks
    ),

    handle(
        R.matchMyAccountRoute,
        () => WA.loadMyAccount
    ),

    handle(
        R.matchProcessLogRoute,
        ({ id }) => WA.loadProcessLog(id)
    ),

    handle(
        R.matchProcessRoute,
        ({ id }) => WA.loadProcess(id)
    ),

    handle(
        R.matchProjectRoute,
        ({ id }) => WA.loadProject(id)
    ),

    handle(
        R.matchRepositoriesRoute,
        () => WA.loadRepositories
    ),

    handle(
        R.matchRootRoute,
        () => navigateToRootProject
    ),

    handle(
        R.matchRunProcessRoute,
        () => WA.loadRunProcess
    ),

    handle(
        R.matchSearchResultsRoute,
        () => WA.loadSearchResults
    ),

    handle(
        R.matchSharedWithMeRoute,
        () => WA.loadSharedWithMe
    ),

    handle(
        R.matchSiteManagerRoute,
        () => WA.loadSiteManager
    ),

    handle(
        R.matchSshKeysAdminRoute,
        () => WA.loadSshKeys
    ),

    handle(
        R.matchSshKeysUserRoute,
        () => WA.loadSshKeys
    ),

    handle(
        R.matchTrashRoute,
        () => WA.loadTrash
    ),

    handle(
        R.matchUsersRoute,
        () => WA.loadUsers
    ),

    handle(
        R.matchAdminVirtualMachineRoute,
        () => WA.loadVirtualMachines
    ),

    handle(
        R.matchUserVirtualMachineRoute,
        () => WA.loadVirtualMachines
    ),

    handle(
        R.matchWorkflowRoute,
        () => WA.loadWorkflow
    ),

];

