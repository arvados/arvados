// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import './index.css';
import { Route } from "react-router";
import createBrowserHistory from "history/createBrowserHistory";
import configureStore from "./store/store";
import { ConnectedRouter } from "react-router-redux";
import ApiToken from "./views-components/api-token/api-token";
import authActions from "./store/auth/auth-action";
import { authService } from "./services/services";
import { getProjectList } from "./store/project/project-action";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from './common/custom-theme';
import CommonResourceService from './common/api/common-resource-service';
import { CollectionResource } from './models/collection';
import { serverApi } from './common/api/server-api';
import { ProcessResource } from './models/process';

const history = createBrowserHistory();

const store = configureStore(history);

store.dispatch(authActions.INIT());
store.dispatch<any>(getProjectList(authService.getUuid()));

// const service = new CommonResourceService<CollectionResource>(serverApi, "collections");
// service.create({ ownerUuid: "qr1hi-j7d0g-u55bcc7fa5w7v4p", name: "Collection 1 short title"});
// service.create({ ownerUuid: "qr1hi-j7d0g-u55bcc7fa5w7v4p", name: "Collection 2 long long long title"});

// const processService = new CommonResourceService<ProcessResource>(serverApi, "container_requests");
// processService.create({ ownerUuid: "qr1hi-j7d0g-u55bcc7fa5w7v4p", name: "Process 1 short title"});
// processService.create({ ownerUuid: "qr1hi-j7d0g-u55bcc7fa5w7v4p", name: "Process 2 long long long title" });

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
