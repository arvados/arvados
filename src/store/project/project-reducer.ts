// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Project } from "../../models/project";
import actions, { ProjectAction } from "./project-action";

export type ProjectState = Project[];

const projectsReducer = (state: ProjectState = [], action: ProjectAction) => {
    return actions.match(action, {
        CREATE_PROJECT: project => [...state, project],
        REMOVE_PROJECT: () => state,
        TOP_PROJECTS_REQUEST: () => state,
        TOP_PROJECTS_SUCCESS: projects => {
            return projects;
        },
        default: () => state
    });
};

export default projectsReducer;
