// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { FormErrors, formValueSelector, initialize, reset, startSubmit, stopSubmit } from "redux-form";
import { RootState } from "store/store";
import { dialogActions } from "store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { ServiceRepository } from "services/services";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { GroupClass } from "models/group";
import { Participant } from "views-components/sharing-dialog/participant-select";
import { ProjectProperties } from "./project-create-actions";
import { getResource } from "store/resources/resources";
import { ProjectResource } from "models/project";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export interface ProjectUpdateFormDialogData {
    uuid: string;
    name: string;
    users?: Participant[];
    description?: string;
    properties?: ProjectProperties;
}

export const PROJECT_UPDATE_FORM_NAME = "projectUpdateFormName";
export const PROJECT_UPDATE_PROPERTIES_FORM_NAME = "projectUpdatePropertiesFormName";
export const PROJECT_UPDATE_FORM_SELECTOR = formValueSelector(PROJECT_UPDATE_FORM_NAME);

export const openProjectUpdateDialog = (resource: ProjectUpdateFormDialogData) => (dispatch: Dispatch, getState: () => RootState) => {
    // Get complete project resource from store to handle consumers passing in partial resources
    const project = getResource<ProjectResource>(resource.uuid)(getState().resources);
    dispatch(initialize(PROJECT_UPDATE_FORM_NAME, project));
    dispatch(
        dialogActions.OPEN_DIALOG({
            id: PROJECT_UPDATE_FORM_NAME,
            data: {
                sourcePanel: GroupClass.PROJECT,
            },
        })
    );
};

export const updateProject =
    (project: ProjectUpdateFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = project.uuid || "";
        dispatch(startSubmit(PROJECT_UPDATE_FORM_NAME));
        try {
            const updatedProject = await services.projectService.update(
                uuid,
                {
                    name: project.name,
                    description: project.description,
                    properties: project.properties,
                },
                false
            );
            dispatch(projectPanelDataActions.REQUEST_ITEMS());
            dispatch(reset(PROJECT_UPDATE_FORM_NAME));
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
            return updatedProject;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME, { name: "Project with the same name already exists." } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
                const errMsg = e.errors ? e.errors.join("") : "There was an error while updating the project";
                dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: errMsg,
                        hideDuration: 2000,
                        kind: SnackbarKind.ERROR,
                    })
                );
            }
            return;
        }
    };
