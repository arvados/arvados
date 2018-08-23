// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createStore, applyMiddleware, compose, Middleware, combineReducers, Store, Action, Dispatch } from 'redux';
import { routerMiddleware, routerReducer } from "react-router-redux";
import thunkMiddleware from 'redux-thunk';
import { History } from "history";

import { projectsReducer } from "./project/project-reducer";
import { authReducer } from "./auth/auth-reducer";
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
import { collectionPanelReducer } from './collection-panel/collection-panel-reducer';
import { dialogReducer } from './dialog/dialog-reducer';
import { collectionsReducer } from './collections/collections-reducer';
import { ServiceRepository } from "~/services/services";
import { treePickerReducer } from './tree-picker/tree-picker-reducer';
import { resourcesReducer } from '~/store/resources/resources-reducer';
import { propertiesReducer } from './properties/properties-reducer';
import { RootState } from './store';

const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
        window && window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__) ||
    compose;

export type RootState = ReturnType<ReturnType<typeof createRootReducer>>;

export type RootStore = Store<RootState, Action> & { dispatch: Dispatch<any> };

export function configureStore(history: History, services: ServiceRepository): RootStore {
    const rootReducer = createRootReducer(services);

    const projectPanelMiddleware = dataExplorerMiddleware(
        new ProjectPanelMiddlewareService(services, PROJECT_PANEL_ID)
    );
    const favoritePanelMiddleware = dataExplorerMiddleware(
        new FavoritePanelMiddlewareService(services, FAVORITE_PANEL_ID)
    );

    const middlewares: Middleware[] = [
        routerMiddleware(history),
        thunkMiddleware.withExtraArgument(services),
        projectPanelMiddleware,
        favoritePanelMiddleware
    ];
    const enhancer = composeEnhancers(applyMiddleware(...middlewares));
    return createStore(rootReducer, enhancer);
}

const createRootReducer = (services: ServiceRepository) => combineReducers({
    auth: authReducer(services),
    projects: projectsReducer,
    collections: collectionsReducer,
    router: routerReducer,
    dataExplorer: dataExplorerReducer,
    collectionPanel: collectionPanelReducer,
    detailsPanel: detailsPanelReducer,
    contextMenu: contextMenuReducer,
    form: formReducer,
    favorites: favoritesReducer,
    snackbar: snackbarReducer,
    collectionPanelFiles: collectionPanelFilesReducer,
    dialog: dialogReducer,
    treePicker: treePickerReducer,
    resources: resourcesReducer,
    properties: propertiesReducer,
});
