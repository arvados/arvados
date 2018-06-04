// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import ProjectList from './components/project-list/project-list';
import './index.css';
import { Route, Router } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import configureStore from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import { TreeItem } from "./components/tree/tree";
import { Project } from "./models/project";

const sampleProjects = [
    [
        'Project 1', [
            ['Project 1.1', [['Project 1.1.1'], ['Project 1.1.2']]],
            ['Project 1.2', [['Project 1.2.1'], ['Project 1.2.2'], ['Project 1.2.3']]]
        ]
    ],
    [
        'Project 2'
    ],
    [
        'Project 3', [['Project 3.1'], ['Project 3.2']]
    ]
];


function buildProjectTree(tree: any[], level = 0): Array<TreeItem<Project>> {
    const projects = tree.map((t, idx) => ({
        id: `l${level}i${idx}${t[0]}`,
        open: false,
        data: {
            name: t[0],
            icon: level === 0 ? <i className="icon-th"/> : <i className="fas fa-folder"/>,
            createdAt: '2018-05-05',
        },
        items: t.length > 1 ? buildProjectTree(t[1], level + 1) : []
    }));
    return projects;
}


const history = createBrowserHistory();
const projects = buildProjectTree(sampleProjects);

const store = configureStore({
    projects,
    router: {
        location: null
    }
}, history);

const App = () =>
    <Provider store={store}>
        <ConnectedRouter history={history}>
            <div>
                <Route path="/" component={Workbench} />
            </div>
        </ConnectedRouter>
    </Provider>;

ReactDOM.render(
    <App />,
    document.getElementById('root') as HTMLElement
);
