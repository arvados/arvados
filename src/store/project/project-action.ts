// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { default as unionize, ofType, UnionOf } from "unionize";

import { Project } from "../../models/project";
import { projectService } from "../../services/services";
import { Dispatch } from "redux";

const actions = unionize({
    CREATE_PROJECT: ofType<Project>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: Project[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>(),
}, {
        tag: 'type',
        value: 'payload'
    });

export const getProjectList = (parentUuid?: string) => (dispatch: Dispatch): Promise<Project[]> => {
    if (parentUuid) {
        dispatch(actions.PROJECTS_REQUEST(parentUuid));
        return projectService.getProjectList(parentUuid).then(projects => {
            dispatch(actions.PROJECTS_SUCCESS({ projects, parentItemId: parentUuid }));
            return projects;
        });
    } return Promise.resolve([]);
};

export type ProjectAction = UnionOf<typeof actions>;
export default actions;
