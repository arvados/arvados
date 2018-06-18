// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import ProjectList from './views-components/project-list/project-list';
import './index.css';
import { Route } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import configureStore from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import ApiToken from "./views-components/api-token/api-token";
import authActions from "./store/auth/auth-action";
import { authService, projectService } from "./services/services";

const history = createBrowserHistory();

const store = configureStore({
    projects: [
    ],
    router: {
        location: null
    },
    auth: {
        user: undefined
    }
}, history);

store.dispatch(authActions.INIT());
const rootUuid = authService.getRootUuid();
store.dispatch<any>(projectService.getProjectList(rootUuid));

const App = () =>
    <Provider store={store}>
        <ConnectedRouter history={history}>
            <div>
                <Route path="/" component={Workbench}/>
                <Route path="/token" component={ApiToken}/>
            </div>
        </ConnectedRouter>
    </Provider>;

ReactDOM.render(
    <App/>,
    document.getElementById('root') as HTMLElement
);
