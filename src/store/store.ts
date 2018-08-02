// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createStore, applyMiddleware, compose, Middleware, combineReducers } from 'redux';
import { routerMiddleware, routerReducer, RouterState } from "react-router-redux";
import thunkMiddleware from 'redux-thunk';
import { History } from "history";

import { projectsReducer, ProjectState } from "./project/project-reducer";
import { sidePanelReducer, SidePanelState } from './side-panel/side-panel-reducer';
import { authReducer, AuthState } from "./auth/auth-reducer";
import { dataExplorerReducer, DataExplorerState } from './data-explorer/data-explorer-reducer';
import { detailsPanelReducer, DetailsPanelState } from './details-panel/details-panel-reducer';
import { contextMenuReducer, ContextMenuState } from './context-menu/context-menu-reducer';
import { reducer as formReducer } from 'redux-form';
import { FavoritesState, favoritesReducer } from './favorites/favorites-reducer';
import { snackbarReducer, SnackbarState } from './snackbar/snackbar-reducer';
import { CollectionPanelFilesState } from './collection-panel/collection-panel-files/collection-panel-files-state';
import { collectionPanelFilesReducer } from './collection-panel/collection-panel-files/collections-panel-files-reducer';
import { dataExplorerMiddleware } from "./data-explorer/data-explorer-middleware";
import { FAVORITE_PANEL_ID } from "./favorite-panel/favorite-panel-action";
import { PROJECT_PANEL_ID } from "./project-panel/project-panel-action";
import { ProjectPanelMiddlewareService } from "./project-panel/project-panel-middleware-service";
import { FavoritePanelMiddlewareService } from "./favorite-panel/favorite-panel-middleware-service";
import { CollectionCreatorState, collectionCreationReducer } from './collections/creator/collection-creator-reducer';
import { CollectionPanelState, collectionPanelReducer } from './collection-panel/collection-panel-reducer';
import { DialogState, dialogReducer } from './dialog/dialog-reducer';

const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
        window && window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__) ||
    compose;

export interface RootState {
    auth: AuthState;
    projects: ProjectState;
    collectionCreation: CollectionCreatorState;
    router: RouterState;
    dataExplorer: DataExplorerState;
    sidePanel: SidePanelState;
    collectionPanel: CollectionPanelState;
    detailsPanel: DetailsPanelState;
    contextMenu: ContextMenuState;
    favorites: FavoritesState;
    snackbar: SnackbarState;
    collectionPanelFiles: CollectionPanelFilesState;
    dialog: DialogState;
}

const rootReducer = combineReducers({
    auth: authReducer,
    projects: projectsReducer,
    collectionCreation: collectionCreationReducer,
    router: routerReducer,
    dataExplorer: dataExplorerReducer,
    sidePanel: sidePanelReducer,
    collectionPanel: collectionPanelReducer,
    detailsPanel: detailsPanelReducer,
    contextMenu: contextMenuReducer,
    form: formReducer,
    favorites: favoritesReducer,
    snackbar: snackbarReducer,
    collectionPanelFiles: collectionPanelFilesReducer,
    dialog: dialogReducer
});

export function configureStore(history: History) {
    const projectPanelMiddleware = dataExplorerMiddleware(
        new ProjectPanelMiddlewareService(PROJECT_PANEL_ID)
    );
    const favoritePanelMiddleware = dataExplorerMiddleware(
        new FavoritePanelMiddlewareService(FAVORITE_PANEL_ID)
    );

    const middlewares: Middleware[] = [
        routerMiddleware(history),
        thunkMiddleware,
        projectPanelMiddleware,
        favoritePanelMiddleware
    ];
    const enhancer = composeEnhancers(applyMiddleware(...middlewares));
    return createStore(rootReducer, enhancer);
}
