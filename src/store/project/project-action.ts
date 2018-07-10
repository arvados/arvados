// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { default as unionize, ofType, UnionOf } from "unionize";

import { Project, ProjectResource } from "../../models/project";
import { projectService } from "../../services/services";
import { Dispatch } from "redux";
import { getResourceKind } from "../../models/resource";
import FilterBuilder from "../../common/api/filter-builder";
import { ThunkAction } from "../../../node_modules/redux-thunk";
import { RootState } from "../store";

const actions = unionize({
    OPEN_PROJECT_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_PROJECT_CREATOR: ofType<{}>(),
    CREATE_PROJECT: ofType<Partial<ProjectResource>>(),
    CREATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    CREATE_PROJECT_ERROR: ofType<string>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: Project[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>()
}, {
        tag: 'type',
        value: 'payload'
    });

export const getProjectList = (parentUuid: string = '') => (dispatch: Dispatch) => {
    dispatch(actions.PROJECTS_REQUEST(parentUuid));
    return projectService.list({
        filters: FilterBuilder
            .create<ProjectResource>()
            .addEqual("ownerUuid", parentUuid)
    }).then(listResults => {
        const projects = listResults.items.map(item => ({
            ...item,
            kind: getResourceKind(item.kind)
        }));
        dispatch(actions.PROJECTS_SUCCESS({ projects, parentItemId: parentUuid }));
        return projects;
    });
};

export const createProject = (project: Partial<ProjectResource>) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { ownerUuid } = getState().projects.creator;
        const projectData = { ownerUuid, ...project };
        dispatch(actions.CREATE_PROJECT(projectData));
        return projectService
            .create(projectData)
            .then(project => dispatch(actions.CREATE_PROJECT_SUCCESS(project)))
            .catch(() => dispatch(actions.CREATE_PROJECT_ERROR("Could not create a project")));
    };

export type ProjectAction = UnionOf<typeof actions>;
export default actions;
