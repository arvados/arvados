// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { Provider } from "react-redux";
import Workbench from './views/workbench/workbench';
import store from "./store/store";
import './index.css';

const App = () =>
    <Provider store={store}>
        <Workbench/>
    </Provider>;

ReactDOM.render(
    <App/>,
    document.getElementById('root') as HTMLElement
);
