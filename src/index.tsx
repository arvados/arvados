// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import ProjectList from './components/project-list/project-list';
import './index.css';
import { Route } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import configureStore from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import ApiToken from "./components/api-token/api-token";
import authActions from "./store/auth/auth-action";
import { projectService } from "./services/services";
import { TreeItem } from "./components/tree/tree";
import { Project } from "./models/project";

function buildProjectTree(tree: any[], level = 0): Array<TreeItem<Project>> {
    const projects = tree.map((t, idx) => ({
        id: `l${level}i${idx}${t[0]}`,
        open: false,
        active: false,
        data: {
            name: t[0],
            icon: level === 0 ? <i className="fas fa-th"/> : <i className="fas fa-folder"/>,
            createdAt: '2018-05-05',
        },
        items: t.length > 1 ? buildProjectTree(t[1], level + 1) : []
    }));
    return projects;
}
const history = createBrowserHistory();
const projects = buildProjectTree(sampleProjects);

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
store.dispatch<any>(projectService.getProjectList());


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
