// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reset, startSubmit, stopSubmit, initialize } from 'redux-form';
import { RootState } from '~/store/store';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { dialogActions } from "~/store/dialog/dialog-actions";
import { projectPanelActions } from '~/store/project-panel/project-panel-action';
import { getProjectList } from '~/store/project/project-action';
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/common/api/common-resource-service';
import { ProjectResource } from '~/models/project';
import { ServiceRepository } from '~/services/services';


export interface ProjectCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
}

export const PROJECT_CREATE_FORM_NAME = 'projectCreateFormName';

export const openProjectCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch) => {
        dispatch(initialize(PROJECT_CREATE_FORM_NAME, { ownerUuid }));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_CREATE_FORM_NAME, data: {} }));
    };

export const addProject = (data: ProjectCreateFormDialogData) =>
    async (dispatch: Dispatch) => {
        await dispatch<any>(createProject(data));
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: "Project has been successfully created.",
            hideDuration: 2000
        }));
    };


const createProject = (project: Partial<ProjectResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROJECT_CREATE_FORM_NAME));
        try {
            const newProject = await services.projectService.create(project);
            dispatch<any>(getProjectList(newProject.ownerUuid));
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_CREATE_FORM_NAME }));
            dispatch(reset(PROJECT_CREATE_FORM_NAME));
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROJECT_CREATE_FORM_NAME, { name: 'Project with the same name already exists.' }));
            }
        }
    };