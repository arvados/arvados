// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createStore, applyMiddleware, compose, Middleware, combineReducers, Store, Action, Dispatch } from 'redux';
import { routerMiddleware, routerReducer } from "react-router-redux";
import thunkMiddleware from 'redux-thunk';
import { History } from "history";
import { handleRedirects } from '../common/redirect-to';

import { authReducer } from "./auth/auth-reducer";
import { authMiddleware } from "./auth/auth-middleware";
import { dataExplorerReducer } from './data-explorer/data-explorer-reducer';
import { detailsPanelReducer } from './details-panel/details-panel-reducer';
import { contextMenuReducer } from './context-menu/context-menu-reducer';
import { reducer as formReducer } from 'redux-form';
import { favoritesReducer } from './favorites/favorites-reducer';
import { snackbarReducer } from './snackbar/snackbar-reducer';
import { collectionPanelFilesReducer } from './collection-panel/collection-panel-files/collection-panel-files-reducer';
import { dataExplorerMiddleware } from "./data-explorer/data-explorer-middleware";
import { FAVORITE_PANEL_ID } from "./favorite-panel/favorite-panel-action";
import { PROJECT_PANEL_ID } from "./project-panel/project-panel-action";
import { ProjectPanelMiddlewareService } from "./project-panel/project-panel-middleware-service";
import { FavoritePanelMiddlewareService } from "./favorite-panel/favorite-panel-middleware-service";
import { AllProcessesPanelMiddlewareService } from "./all-processes-panel/all-processes-panel-middleware-service";
import { collectionPanelReducer } from './collection-panel/collection-panel-reducer';
import { dialogReducer } from './dialog/dialog-reducer';
import { ServiceRepository } from "services/services";
import { treePickerReducer } from './tree-picker/tree-picker-reducer';
import { resourcesReducer } from 'store/resources/resources-reducer';
import { propertiesReducer } from './properties/properties-reducer';
import { fileUploaderReducer } from './file-uploader/file-uploader-reducer';
import { TrashPanelMiddlewareService } from "store/trash-panel/trash-panel-middleware-service";
import { TRASH_PANEL_ID } from "store/trash-panel/trash-panel-action";
import { processLogsPanelReducer } from './process-logs-panel/process-logs-panel-reducer';
import { processPanelReducer } from 'store/process-panel/process-panel-reducer';
import { SHARED_WITH_ME_PANEL_ID } from 'store/shared-with-me-panel/shared-with-me-panel-actions';
import { SharedWithMeMiddlewareService } from './shared-with-me-panel/shared-with-me-middleware-service';
import { progressIndicatorReducer } from './progress-indicator/progress-indicator-reducer';
import { runProcessPanelReducer } from 'store/run-process-panel/run-process-panel-reducer';
import { WorkflowMiddlewareService } from './workflow-panel/workflow-middleware-service';
import { WORKFLOW_PANEL_ID } from './workflow-panel/workflow-panel-actions';
import { appInfoReducer } from 'store/app-info/app-info-reducer';
import { searchBarReducer } from './search-bar/search-bar-reducer';
import { SEARCH_RESULTS_PANEL_ID } from 'store/search-results-panel/search-results-panel-actions';
import { SearchResultsMiddlewareService } from './search-results-panel/search-results-middleware-service';
import { virtualMachinesReducer } from "store/virtual-machines/virtual-machines-reducer";
import { repositoriesReducer } from 'store/repositories/repositories-reducer';
import { keepServicesReducer } from 'store/keep-services/keep-services-reducer';
import { UserMiddlewareService } from 'store/users/user-panel-middleware-service';
import { USERS_PANEL_ID } from 'store/users/users-actions';
import { UserProfileGroupsMiddlewareService } from 'store/user-profile/user-profile-groups-middleware-service';
import { USER_PROFILE_PANEL_ID } from 'store/user-profile/user-profile-actions'
import { GroupsPanelMiddlewareService } from 'store/groups-panel/groups-panel-middleware-service';
import { GROUPS_PANEL_ID } from 'store/groups-panel/groups-panel-actions';
import { GroupDetailsPanelMembersMiddlewareService } from 'store/group-details-panel/group-details-panel-members-middleware-service';
import { GroupDetailsPanelPermissionsMiddlewareService } from 'store/group-details-panel/group-details-panel-permissions-middleware-service';
import { GROUP_DETAILS_MEMBERS_PANEL_ID, GROUP_DETAILS_PERMISSIONS_PANEL_ID } from 'store/group-details-panel/group-details-panel-actions';
import { LINK_PANEL_ID } from 'store/link-panel/link-panel-actions';
import { LinkMiddlewareService } from 'store/link-panel/link-panel-middleware-service';
import { API_CLIENT_AUTHORIZATION_PANEL_ID } from 'store/api-client-authorizations/api-client-authorizations-actions';
import { ApiClientAuthorizationMiddlewareService } from 'store/api-client-authorizations/api-client-authorizations-middleware-service';
import { PublicFavoritesMiddlewareService } from 'store/public-favorites-panel/public-favorites-middleware-service';
import { PUBLIC_FAVORITE_PANEL_ID } from 'store/public-favorites-panel/public-favorites-action';
import { publicFavoritesReducer } from 'store/public-favorites/public-favorites-reducer';
import { linkAccountPanelReducer } from './link-account-panel/link-account-panel-reducer';
import { CollectionsWithSameContentAddressMiddlewareService } from 'store/collections-content-address-panel/collections-content-address-middleware-service';
import { COLLECTIONS_CONTENT_ADDRESS_PANEL_ID } from 'store/collections-content-address-panel/collections-content-address-panel-actions';
import { ownerNameReducer } from 'store/owner-name/owner-name-reducer';
import { SubprocessMiddlewareService } from 'store/subprocess-panel/subprocess-panel-middleware-service';
import { SUBPROCESS_PANEL_ID } from 'store/subprocess-panel/subprocess-panel-actions';
import { ALL_PROCESSES_PANEL_ID } from './all-processes-panel/all-processes-panel-action';
import { Config } from 'common/config';
import { pluginConfig } from 'plugins';
import { MiddlewareListReducer } from 'common/plugintypes';
import { sidePanelReducer } from './side-panel/side-panel-reducer'

declare global {
    interface Window {
      __REDUX_DEVTOOLS_EXTENSION_COMPOSE__?: typeof compose;
    }
}

const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
        window && window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__) ||
    compose;

export type RootState = ReturnType<ReturnType<typeof createRootReducer>>;

export type RootStore = Store<RootState, Action> & { dispatch: Dispatch<any> };

export function configureStore(history: History, services: ServiceRepository, config: Config): RootStore {
    const rootReducer = createRootReducer(services);

    const projectPanelMiddleware = dataExplorerMiddleware(
        new ProjectPanelMiddlewareService(services, PROJECT_PANEL_ID)
    );
    const favoritePanelMiddleware = dataExplorerMiddleware(
        new FavoritePanelMiddlewareService(services, FAVORITE_PANEL_ID)
    );
    const allProcessessPanelMiddleware = dataExplorerMiddleware(
        new AllProcessesPanelMiddlewareService(services, ALL_PROCESSES_PANEL_ID)
    );
    const trashPanelMiddleware = dataExplorerMiddleware(
        new TrashPanelMiddlewareService(services, TRASH_PANEL_ID)
    );
    const searchResultsPanelMiddleware = dataExplorerMiddleware(
        new SearchResultsMiddlewareService(services, SEARCH_RESULTS_PANEL_ID)
    );
    const sharedWithMePanelMiddleware = dataExplorerMiddleware(
        new SharedWithMeMiddlewareService(services, SHARED_WITH_ME_PANEL_ID)
    );
    const workflowPanelMiddleware = dataExplorerMiddleware(
        new WorkflowMiddlewareService(services, WORKFLOW_PANEL_ID)
    );
    const userPanelMiddleware = dataExplorerMiddleware(
        new UserMiddlewareService(services, USERS_PANEL_ID)
    );
    const userProfileGroupsMiddleware = dataExplorerMiddleware(
        new UserProfileGroupsMiddlewareService(services, USER_PROFILE_PANEL_ID)
    );
    const groupsPanelMiddleware = dataExplorerMiddleware(
        new GroupsPanelMiddlewareService(services, GROUPS_PANEL_ID)
    );
    const groupDetailsPanelMembersMiddleware = dataExplorerMiddleware(
        new GroupDetailsPanelMembersMiddlewareService(services, GROUP_DETAILS_MEMBERS_PANEL_ID)
    );
    const groupDetailsPanelPermissionsMiddleware = dataExplorerMiddleware(
        new GroupDetailsPanelPermissionsMiddlewareService(services, GROUP_DETAILS_PERMISSIONS_PANEL_ID)
    );
    const linkPanelMiddleware = dataExplorerMiddleware(
        new LinkMiddlewareService(services, LINK_PANEL_ID)
    );
    const apiClientAuthorizationMiddlewareService = dataExplorerMiddleware(
        new ApiClientAuthorizationMiddlewareService(services, API_CLIENT_AUTHORIZATION_PANEL_ID)
    );
    const publicFavoritesMiddleware = dataExplorerMiddleware(
        new PublicFavoritesMiddlewareService(services, PUBLIC_FAVORITE_PANEL_ID)
    );
    const collectionsContentAddress = dataExplorerMiddleware(
        new CollectionsWithSameContentAddressMiddlewareService(services, COLLECTIONS_CONTENT_ADDRESS_PANEL_ID)
    );
    const subprocessMiddleware = dataExplorerMiddleware(
        new SubprocessMiddlewareService(services, SUBPROCESS_PANEL_ID)
    );
    const redirectToMiddleware = (store: any) => (next: any) => (action: any) => {
        const state = store.getState();

        if (state.auth && state.auth.apiToken) {
            handleRedirects(state.auth.apiToken, config);
        }

        return next(action);
    };

    let middlewares: Middleware[] = [
        routerMiddleware(history),
        thunkMiddleware.withExtraArgument(services),
        authMiddleware(services),
        projectPanelMiddleware,
        favoritePanelMiddleware,
        allProcessessPanelMiddleware,
        trashPanelMiddleware,
        searchResultsPanelMiddleware,
        sharedWithMePanelMiddleware,
        workflowPanelMiddleware,
        userPanelMiddleware,
        userProfileGroupsMiddleware,
        groupsPanelMiddleware,
        groupDetailsPanelMembersMiddleware,
        groupDetailsPanelPermissionsMiddleware,
        linkPanelMiddleware,
        apiClientAuthorizationMiddlewareService,
        publicFavoritesMiddleware,
        collectionsContentAddress,
        subprocessMiddleware,
    ];

    const reduceMiddlewaresFn: (a: Middleware[],
        b: MiddlewareListReducer) => Middleware[] = (a, b) => b(a, services);

    middlewares = pluginConfig.middlewares.reduce(reduceMiddlewaresFn, middlewares);

    const enhancer = composeEnhancers(applyMiddleware(redirectToMiddleware, ...middlewares));
    return createStore(rootReducer, enhancer);
}

const createRootReducer = (services: ServiceRepository) => combineReducers({
    auth: authReducer(services),
    collectionPanel: collectionPanelReducer,
    collectionPanelFiles: collectionPanelFilesReducer,
    contextMenu: contextMenuReducer,
    dataExplorer: dataExplorerReducer,
    detailsPanel: detailsPanelReducer,
    dialog: dialogReducer,
    favorites: favoritesReducer,
    ownerName: ownerNameReducer,
    publicFavorites: publicFavoritesReducer,
    form: formReducer,
    processLogsPanel: processLogsPanelReducer,
    properties: propertiesReducer,
    resources: resourcesReducer,
    router: routerReducer,
    snackbar: snackbarReducer,
    treePicker: treePickerReducer,
    fileUploader: fileUploaderReducer,
    processPanel: processPanelReducer,
    progressIndicator: progressIndicatorReducer,
    runProcessPanel: runProcessPanelReducer,
    appInfo: appInfoReducer,
    searchBar: searchBarReducer,
    virtualMachines: virtualMachinesReducer,
    repositories: repositoriesReducer,
    keepServices: keepServicesReducer,
    linkAccountPanel: linkAccountPanelReducer,
    sidePanel: sidePanelReducer
});
