// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import ReactDOM from 'react-dom';
import { WorkbenchPanel } from './workbench';
import { Provider } from "react-redux";
import { configureStore } from "store/store";
import { createBrowserHistory } from "history";
import { ConnectedRouter } from "connected-react-router";
import { ThemeProvider, StyledEngineProvider } from '@mui/material/styles';
import { CustomTheme } from 'common/custom-theme';
import { createServices } from "services/services";

const history = createBrowserHistory();

it('renders without crashing', () => {
    const div = document.createElement('div');
    const services = createServices("/arvados/v1");
	services.authService.getUuid = cy.stub().returns('test');
    const store = configureStore(createBrowserHistory(), services);
    ReactDOM.render(
        <StyledEngineProvider injectFirst>
            <ThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <ConnectedRouter history={history}>
                        <WorkbenchPanel />
                    </ConnectedRouter>
                </Provider>
            </ThemeProvider>
        </StyledEngineProvider>,
    div);
    ReactDOM.unmountComponentAtNode(div);
});
