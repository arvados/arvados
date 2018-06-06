// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import ProjectList from './components/project-list/project-list';
import './index.css';
import { Redirect, Route, RouteProps, Router, RouterProps } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import configureStore from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import ApiToken from "./components/api-token/api-token";

const history = createBrowserHistory();
const store = configureStore({
    projects: [
        { name: 'Mouse genome', createdAt: '2018-05-01' },
        { name: 'Human body', createdAt: '2018-05-01' },
        { name: 'Secret operation', createdAt: '2018-05-01' }
    ],
    router: {
        location: null
    },
    auth: {
        user: undefined
    }
}, history);

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
