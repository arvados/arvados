// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import ReactDOM from 'react-dom';
import { WorkbenchPanel } from './workbench';
import { Provider } from "react-redux";
import { configureStore } from "store/store";
import { createBrowserHistory } from "history";
import { ConnectedRouter } from "react-router-redux";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from 'common/custom-theme';
import { createServices } from "services/services";
import 'jest-localstorage-mock';

const history = createBrowserHistory();

it('renders without crashing', () => {
    const div = document.createElement('div');
    const services = createServices("/arvados/v1");
	services.authService.getUuid = jest.fn().mockReturnValueOnce('test');
    const store = configureStore(createBrowserHistory(), services);
    ReactDOM.render(
        <MuiThemeProvider theme={CustomTheme}>
            <Provider store={store}>
                <ConnectedRouter history={history}>
                    <WorkbenchPanel />
                </ConnectedRouter>
            </Provider>
        </MuiThemeProvider>,
    div);
    ReactDOM.unmountComponentAtNode(div);
});
