// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { default as unionize, ofType, UnionOf } from "unionize";

import { ProjectResource } from "~/models/project";
import { Dispatch } from "redux";
import { FilterBuilder } from "~/common/api/filter-builder";
import { RootState } from "../store";
import { checkPresenceInFavorites } from "../favorites/favorites-actions";
import { ServiceRepository } from "~/services/services";

export const projectActions = unionize({
    OPEN_PROJECT_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_PROJECT_CREATOR: ofType<{}>(),
    CREATE_PROJECT: ofType<Partial<ProjectResource>>(),
    CREATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: ProjectResource[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>()
}, {
    tag: 'type',
    value: 'payload'
});

export const getProjectList = (parentUuid: string = '') => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(projectActions.PROJECTS_REQUEST(parentUuid));
        return services.projectService.list({
            filters: FilterBuilder
                .create()
                .addEqual("ownerUuid", parentUuid)
        }).then(({ items: projects }) => {
            dispatch(projectActions.PROJECTS_SUCCESS({ projects, parentItemId: parentUuid }));
            dispatch<any>(checkPresenceInFavorites(projects.map(project => project.uuid)));
            return projects;
        });
    };

export const createProject = (project: Partial<ProjectResource>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { ownerUuid } = getState().projects.creator;
        const projectData = { ownerUuid, ...project };
        dispatch(projectActions.CREATE_PROJECT(projectData));
        return services.projectService
            .create(projectData)
            .then(project => dispatch(projectActions.CREATE_PROJECT_SUCCESS(project)));
    };

export type ProjectAction = UnionOf<typeof projectActions>;
