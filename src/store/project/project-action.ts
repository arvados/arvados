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
import { projectPanelActions } from "~/store/project-panel/project-panel-action";
import { updateDetails } from "~/store/details-panel/details-panel-action";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { trashPanelActions } from "~/store/trash-panel/trash-panel-action";
import { sidePanelActions } from "~/store/side-panel/side-panel-action";
import { SidePanelId } from "~/store/side-panel/side-panel-reducer";

export const projectActions = unionize({
    OPEN_PROJECT_CREATOR: ofType<{ ownerUuid: string }>(),
    CLOSE_PROJECT_CREATOR: ofType<{}>(),
    CREATE_PROJECT: ofType<Partial<ProjectResource>>(),
    CREATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    OPEN_PROJECT_UPDATER: ofType<{ uuid: string}>(),
    CLOSE_PROJECT_UPDATER: ofType<{}>(),
    UPDATE_PROJECT_SUCCESS: ofType<ProjectResource>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<string>(),
    PROJECTS_SUCCESS: ofType<{ projects: ProjectResource[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<{ itemId: string, open?: boolean, recursive?: boolean }>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<{ itemId: string, active?: boolean, recursive?: boolean }>(),
    RESET_PROJECT_TREE_ACTIVITY: ofType<string>()
}, {
    tag: 'type',
    value: 'payload'
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

export const updateProject = (project: Partial<ProjectResource>) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { uuid } = getState().projects.updater;
        return services.projectService
            .update(uuid, project)
            .then(project => {
                dispatch(projectActions.UPDATE_PROJECT_SUCCESS(project));
                dispatch(projectPanelActions.REQUEST_ITEMS());
                dispatch<any>(getProjectList(project.ownerUuid));
                dispatch<any>(updateDetails(project));
            });
    };

export const toggleProjectTrashed = (resource: { uuid: string; name: string, isTrashed?: boolean, ownerUuid?: string }) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<any> => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Working..." }));
        if (resource.isTrashed) {
            return services.groupsService.untrash(resource.uuid).then(() => {
                dispatch<any>(getProjectList(resource.ownerUuid)).then(() => {
                    dispatch(sidePanelActions.TOGGLE_SIDE_PANEL_ITEM_OPEN(SidePanelId.PROJECTS));
                    dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN({ itemId: resource.ownerUuid!!, open: true, recursive: true }));
                });
                dispatch(trashPanelActions.REQUEST_ITEMS());
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Restored from trash",
                    hideDuration: 2000
                }));
            });
        } else {
            return services.groupsService.trash(resource.uuid).then(() => {
                dispatch<any>(getProjectList(resource.ownerUuid)).then(() => {
                    dispatch(projectActions.TOGGLE_PROJECT_TREE_ITEM_OPEN({ itemId: resource.ownerUuid!!, open: true, recursive: true }));
                });
                dispatch(snackbarActions.CLOSE_SNACKBAR());
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Added to trash",
                    hideDuration: 2000
                }));
            });
        }
    };

export type ProjectAction = UnionOf<typeof projectActions>;
