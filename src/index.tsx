// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import { Workbench } from './views/workbench/workbench';
import './index.css';
import { Route } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import { configureStore } from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import { ApiToken } from "./views-components/api-token/api-token";
import { initAuth } from "./store/auth/auth-action";
import { createServices } from "./services/services";
import { getProjectList } from "./store/project/project-action";
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
import { initPickerProjectTree } from './views-components/project-tree-picker/project-tree-picker';

addMenuActionSet(ContextMenuKind.ROOT_PROJECT, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.PROJECT, projectActionSet);
addMenuActionSet(ContextMenuKind.RESOURCE, resourceActionSet);
addMenuActionSet(ContextMenuKind.FAVORITE, favoriteActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES, collectionFilesActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_FILES_ITEM, collectionFilesItemActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION, collectionActionSet);
addMenuActionSet(ContextMenuKind.COLLECTION_RESOURCE, collectionResourceActionSet);

fetchConfig()
    .then(config => {
        const history = createBrowserHistory();
        const services = createServices(config);
        const store = configureStore(history, services);

        store.dispatch(initAuth());
        store.dispatch(getProjectList(services.authService.getUuid()));
        store.dispatch(initPickerProjectTree());
        
        const TokenComponent = (props: any) => <ApiToken authService={services.authService} {...props}/>;
        const WorkbenchComponent = (props: any) => <Workbench authService={services.authService} {...props}/>;

        const App = () =>
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <div>
                            <Route path="/" component={WorkbenchComponent} />
                            <Route path="/token" component={TokenComponent} />
                        </div>
                    </ConnectedRouter>
                </Provider>
            </MuiThemeProvider>;

        ReactDOM.render(
            <App />,
            document.getElementById('root') as HTMLElement
        );
    });


