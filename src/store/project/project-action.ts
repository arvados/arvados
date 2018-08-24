// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from '~/common/unionize';
import { ProjectResource } from "~/models/project";
import { Dispatch } from "redux";
import { FilterBuilder } from "~/common/api/filter-builder";
import { RootState } from "../store";
import { updateFavorites } from "../favorites/favorites-actions";
import { ServiceRepository } from "~/services/services";
import { projectPanelActions } from "~/store/project-panel/project-panel-action";
import { resourcesActions } from "~/store/resources/resources-actions";
import { reset } from 'redux-form';

export const projectActions = unionize({
    OPEN_PROJECT_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_PROJECT_CREATOR: ofType<{}>(),
    CREATE_PROJECT: ofType<Partial<ProjectResource>>(),
    CREATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    OPEN_PROJECT_UPDATER: ofType<{ uuid: string }>(),
    CLOSE_PROJECT_UPDATER: ofType<{}>(),
    UPDATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: ProjectResource[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>()
});

export const PROJECT_FORM_NAME = 'projectEditDialog';

export const getProjectList = (parentUuid: string = '') =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(projectActions.PROJECTS_REQUEST(parentUuid));
        return services.projectService.list({
            filters: new FilterBuilder()
                .addEqual("ownerUuid", parentUuid)
                .getFilters()
        }).then(({ items: projects }) => {
            dispatch(projectActions.PROJECTS_SUCCESS({ projects, parentItemId: parentUuid }));
            dispatch<any>(updateFavorites(projects.map(project => project.uuid)));
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

export const updateProject = (project: Partial<ProjectResource>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { uuid } = getState().projects.updater;
        return services.projectService
            .update(uuid, project)
            .then(project => {
                dispatch(projectActions.UPDATE_PROJECT_SUCCESS(project));
                dispatch(projectPanelActions.REQUEST_ITEMS());
                dispatch<any>(getProjectList(project.ownerUuid));
                dispatch(resourcesActions.SET_RESOURCES([project]));
            });
    };

export const PROJECT_CREATE_DIALOG = "projectCreateDialog";

export const openProjectCreator = (ownerUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(reset(PROJECT_CREATE_DIALOG));
        dispatch(projectActions.OPEN_PROJECT_CREATOR({ ownerUuid }));
    };

export type ProjectAction = UnionOf<typeof projectActions>;
