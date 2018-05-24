// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { combineReducers } from "redux";
import { StateType } from "typesafe-actions";
import projectsReducer from "./project-reducer";
import { routerReducer } from "react-router-redux";

const rootReducer = combineReducers({
    projects: projectsReducer,
    router: routerReducer
});

export type RootState = StateType<typeof rootReducer>;

export default rootReducer;
