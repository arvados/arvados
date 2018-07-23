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
import { authActions } from "./store/auth/auth-action";
import { authService } from "./services/services";
import { getProjectList } from "./store/project/project-action";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from './common/custom-theme';
import { fetchConfig } from './common/config';
import { setBaseUrl } from './common/api/server-api';
import { addMenuActionSet, ContextMenuKind } from "./views-components/context-menu/context-menu";
import { rootProjectActionSet } from "./views-components/context-menu/action-sets/root-project-action-set";
import { projectActionSet } from "./views-components/context-menu/action-sets/project-action-set";

addMenuActionSet(ContextMenuKind.RootProject, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.Project, projectActionSet);

fetchConfig()
    .then(config => {

        setBaseUrl(config.API_HOST);

        const history = createBrowserHistory();
        const store = configureStore(history);

        store.dispatch(authActions.INIT());
        store.dispatch<any>(getProjectList(authService.getUuid()));

        const App = () =>
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <div>
                            <Route path="/" component={Workbench} />
                            <Route path="/token" component={ApiToken} />
                        </div>
                    </ConnectedRouter>
                </Provider>
            </MuiThemeProvider>;

        ReactDOM.render(
            <App />,
            document.getElementById('root') as HTMLElement
        );
    });


