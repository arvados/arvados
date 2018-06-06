// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from "redux";
import { routerReducer, RouterState } from "react-router-redux";
import authReducer, { AuthState } from "./auth-reducer";
import projectsReducer, { ProjectState } from "./project-reducer";

export interface RootState {
    auth: AuthState,
    projects: ProjectState,
    router: RouterState
}

const rootReducer = combineReducers({
    auth: authReducer,
    projects: projectsReducer,
    router: routerReducer
});

export default rootReducer;
