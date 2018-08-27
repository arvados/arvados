// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { initialize, startSubmit, stopSubmit } from 'redux-form';
import { RootState } from "~/store/store";
import { loadDetailsPanel } from "~/store/details-panel/details-panel-action";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { ContextMenuResource } from '~/store/context-menu/context-menu-reducer';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/common/api/common-resource-service";
import { ServiceRepository } from "~/services/services";
import { ProjectResource } from '~/models/project';
import { getProjectList } from '~/store/project/project-action';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';

export interface ProjectUpdateFormDialogData {
    uuid: string;
    name: string;
    description: string;
}

export const PROJECT_UPDATE_FORM_NAME = 'projectUpdateFormName';

export const openProjectUpdateDialog = (resource: ContextMenuResource) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(PROJECT_UPDATE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_UPDATE_FORM_NAME, data: {} }));
    };

export const editProject = (data: ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch) => {
        await dispatch<any>(updateProject(data));
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: "Project has been successfully updated.",
            hideDuration: 2000
        }));
    };

export const updateProject = (project: Partial<ProjectResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = project.uuid || '';
        dispatch(startSubmit(PROJECT_UPDATE_FORM_NAME));
        try {
            const updatedProject = await services.projectService.update(uuid, project);
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch<any>(getProjectList(updatedProject.ownerUuid));
            dispatch<any>(loadDetailsPanel(updatedProject.uuid));
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME, { name: 'Project with the same name already exists.' }));
            }
        }
    };