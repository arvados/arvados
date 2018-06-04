// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from "redux";
import { StateType } from "typesafe-actions";
import { routerReducer } from "react-router-redux";
import authReducer from "./auth-reducer";
import projectsReducer from "./project-reducer";

const rootReducer = combineReducers({
    auth: authReducer,
    projects: projectsReducer,
    router: routerReducer
});

export type RootState = StateType<typeof rootReducer>;

export default rootReducer;
