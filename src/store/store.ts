// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createStore, applyMiddleware, compose, Middleware } from 'redux';
import { default as rootReducer, RootState } from "./root-reducer";
import { routerMiddleware } from "react-router-redux";
import { History } from "history";

const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
    window && window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__) ||
    compose;

export default function configureStore(initialState: RootState, history: History) {
    const middlewares: Middleware[] = [
        routerMiddleware(history)
    ];
    const enhancer = composeEnhancers(applyMiddleware(...middlewares));
    return createStore(rootReducer, initialState!, enhancer);
}
