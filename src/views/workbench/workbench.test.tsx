// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import Workbench from '../../views/workbench/workbench';
import { Provider } from "react-redux";
import store from "../../store/store";

it('renders without crashing', () => {
    const div = document.createElement('div');
    ReactDOM.render(<Provider store={store}><Workbench/></Provider>, div);
    ReactDOM.unmountComponentAtNode(div);
});
