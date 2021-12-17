// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import {
    change,
    FormErrors,
    formValueSelector,
    initialize,
    reset,
    startSubmit,
    stopSubmit
} from 'redux-form';
import { RootState } from "store/store";
import { dialogActions } from "store/dialog/dialog-actions";
import {
    getCommonResourceServiceError,
    CommonResourceServiceError
} from "services/common-service/common-resource-service";
import { ServiceRepository } from "services/services";
import { projectPanelActions } from 'store/project-panel/project-panel-action';
import { GroupClass } from "models/group";
import { Participant } from "views-components/sharing-dialog/participant-select";
import { ResourcePropertiesFormData } from "views-components/resource-properties-form/resource-properties-form";
import { addProperty, deleteProperty } from "lib/resource-properties";
import { ProjectProperties } from "./project-create-actions";

export interface ProjectUpdateFormDialogData {
    uuid: string;
    name: string;
    users?: Participant[];
    description?: string;
    properties?: ProjectProperties;
}

export const PROJECT_UPDATE_FORM_NAME = 'projectUpdateFormName';
export const PROJECT_UPDATE_PROPERTIES_FORM_NAME = 'projectUpdatePropertiesFormName';
export const PROJECT_UPDATE_FORM_SELECTOR = formValueSelector(PROJECT_UPDATE_FORM_NAME);

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
            const updatedProject = await services.projectService.update(
                uuid,
                {
                    name: project.name,
                    description: project.description,
                    properties: project.properties,
                });
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

export const addPropertyToUpdateProjectForm = (data: ResourcePropertiesFormData) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const properties = { ...PROJECT_UPDATE_FORM_SELECTOR(getState(), 'properties') };
        const key = data.keyID || data.key;
        const value =  data.valueID || data.value;
        dispatch(change(
            PROJECT_UPDATE_FORM_NAME,
            'properties',
            addProperty(properties, key, value)));
    };

export const removePropertyFromUpdateProjectForm = (key: string, value: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const properties = { ...PROJECT_UPDATE_FORM_SELECTOR(getState(), 'properties') };
        dispatch(change(
            PROJECT_UPDATE_FORM_NAME,
            'properties',
            deleteProperty(properties, key, value)));
    };
