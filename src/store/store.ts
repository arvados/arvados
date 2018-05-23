// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createStore, applyMiddleware, compose, Middleware } from 'redux';
import { default as rootReducer, RootState } from "./root-reducer";

const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
    window && window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__) ||
    compose;

function configureStore(initialState?: RootState) {
    const middlewares: Middleware[] = [];
    const enhancer = composeEnhancers(applyMiddleware(...middlewares));
    return createStore(rootReducer, initialState!, enhancer);
}

const store = configureStore({
    projects: [
        { name: 'Mouse genome', createdAt: '2018-05-01' },
        { name: 'Human body', createdAt: '2018-05-01' },
        { name: 'Secret operation', createdAt: '2018-05-01' }
    ]
});

export default store;
