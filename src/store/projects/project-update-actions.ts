// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { FormErrors, initialize, reset, startSubmit, stopSubmit } from 'redux-form';
import { RootState } from "store/store";
import { dialogActions } from "store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { ServiceRepository } from "services/services";
import { projectPanelActions } from 'store/project-panel/project-panel-action';
import { GroupClass } from "models/group";

export interface ProjectUpdateFormDialogData {
    uuid: string;
    name: string;
    description?: string;
}

export const PROJECT_UPDATE_FORM_NAME = 'projectUpdateFormName';

export const openProjectUpdateDialog = (resource: ProjectUpdateFormDialogData) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(initialize(PROJECT_UPDATE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_UPDATE_FORM_NAME, data: {sourcePanel: GroupClass.PROJECT} }));
    };

export const updateProject = (project: ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = project.uuid || '';
        dispatch(startSubmit(PROJECT_UPDATE_FORM_NAME));
        try {
            const updatedProject = await services.projectService.update(uuid, { name: project.name, description: project.description });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(reset(PROJECT_UPDATE_FORM_NAME));
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
            return updatedProject;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME, { name: 'Project with the same name already exists.' } as FormErrors));
            }
            return ;
        }
    };
