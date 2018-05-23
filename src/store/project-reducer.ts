// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { getType } from "typesafe-actions";
import { Project } from "../models/project";
import { actions, ProjectAction } from "./project-action";

type ProjectState = Project[];

const projectsReducer = (state: ProjectState = [], action: ProjectAction) => {
    switch (action.type) {
        case getType(actions.createProject): {
            return [...state, action.payload];
        }
        default:
            return state;
    }
};

export default projectsReducer;
