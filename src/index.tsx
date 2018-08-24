// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import { Workbench } from './views/workbench/workbench';
import './index.css';
import { Route } from 'react-router';
import createBrowserHistory from "history/createBrowserHistory";
import { History } from "history";
import { configureStore, RootStore } from './store/store';
import { ConnectedRouter } from "react-router-redux";
import { ApiToken } from "./views-components/api-token/api-token";
import { initAuth } from "./store/auth/auth-action";
import { createServices } from "./services/services";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from './common/custom-theme';
import { fetchConfig } from './common/config';
import { addMenuActionSet, ContextMenuKind } from "./views-components/context-menu/context-menu";
import { rootProjectActionSet } from "./views-components/context-menu/action-sets/root-project-action-set";
import { projectActionSet } from "./views-components/context-menu/action-sets/project-action-set";
import { resourceActionSet } from './views-components/context-menu/action-sets/resource-action-set';
import { favoriteActionSet } from "./views-components/context-menu/action-sets/favorite-action-set";
import { collectionFilesActionSet } from './views-components/context-menu/action-sets/collection-files-action-set';
import { collectionFilesItemActionSet } from './views-components/context-menu/action-sets/collection-files-item-action-set';
import { collectionActionSet } from './views-components/context-menu/action-sets/collection-action-set';
import { collectionResourceActionSet } from './views-components/context-menu/action-sets/collection-resource-action-set';
import { addRouteChangeHandlers } from './routes/routes';
import { loadWorkbench } from './store/navigation/navigation-action';

const getBuildNumber = () => "BN-" + (process.env.REACT_APP_BUILD_NUMBER || "dev");
const getGitCommit = () => "GIT-" + (process.env.REACT_APP_GIT_COMMIT || "latest").substr(0, 7);
const getBuildInfo = () => getBuildNumber() + " / " + getGitCommit();

const buildInfo = getBuildInfo();

console.log(`Starting arvados [${buildInfo}]`);

addMenuActionSet(ContextMenuKind.ROOT_PROJECT, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.PROJECT, projectActionSet);
addMenuActionSet(ContextMenuKind.RESOURCE, resourceActionSet);
addMenuActionSet(ContextMenuKind.FAVORITE, favoriteActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES, collectionFilesActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES_ITEM, collectionFilesItemActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION, collectionActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_RESOURCE, collectionResourceActionSet);

fetchConfig()
    .then(async (config) => {
        const history = createBrowserHistory();
        const services = createServices(config);
        const store = configureStore(history, services);

        store.subscribe(initListener(history, store));

        store.dispatch(initAuth());

        const TokenComponent = (props: any) => <ApiToken authService={services.authService} {...props} />;
        const WorkbenchComponent = (props: any) => <Workbench authService={services.authService} buildInfo={buildInfo} {...props} />;

        const App = () =>
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <div>
                            <Route path="/token" component={TokenComponent} />
                            <Route path="/" component={WorkbenchComponent} />
                        </div>
                    </ConnectedRouter>
                </Provider>
            </MuiThemeProvider>;

        ReactDOM.render(
            <App />,
            document.getElementById('root') as HTMLElement
        );


    });

const initListener = (history: History, store: RootStore) => {
    let initialized = false;
    return async () => {
        const { router, auth } = store.getState();
        if (router.location && auth.user && !initialized) {
            initialized = true;
            await store.dispatch(loadWorkbench());
            addRouteChangeHandlers(history, store);
        }
    };
};


