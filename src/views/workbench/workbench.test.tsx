// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Workbench } from '../../views/workbench/workbench';
import { Provider } from "react-redux";
import { configureStore } from "../../store/store";
import createBrowserHistory from "history/createBrowserHistory";
import { ConnectedRouter } from "react-router-redux";
import { MuiThemeProvider } from '@material-ui/core/styles';
import { CustomTheme } from '../../common/custom-theme';
import { createServices } from "../../services/services";
import { AuthService } from "../../services/auth-service/auth-service";
import Axios from "axios";

const history = createBrowserHistory();
const authService = new AuthService(Axios.create(), '/arvados/v1');

authService.getUuid = jest.fn().mockReturnValueOnce('test');

it('renders without crashing', () => {
    const div = document.createElement('div');
    const store = configureStore(createBrowserHistory(), createServices("/arvados/v1"));
    ReactDOM.render(
        <MuiThemeProvider theme={CustomTheme}>
            <Provider store={store}>
                <ConnectedRouter history={history}>
                    <Workbench authService={authService} />
                </ConnectedRouter>
            </Provider>
        </MuiThemeProvider>,
    div);
    ReactDOM.unmountComponentAtNode(div);
});
