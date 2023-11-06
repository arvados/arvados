// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import {
    reset,
    startSubmit,
    stopSubmit,
    initialize,
    FormErrors,
    formValueSelector
} from 'redux-form';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { dialogActions } from "store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from 'services/common-service/common-resource-service';
import { ProjectResource } from 'models/project';
import { ServiceRepository } from 'services/services';
import { matchProjectRoute, matchRunProcessRoute } from 'routes/routes';
import { RouterState } from "react-router-redux";
import { GroupClass } from "models/group";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export interface ProjectCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
    properties: ProjectProperties;
}

export interface ProjectProperties {
    [key: string]: string | string[];
}

export const PROJECT_CREATE_FORM_NAME = 'projectCreateFormName';
export const PROJECT_CREATE_PROPERTIES_FORM_NAME = 'projectCreatePropertiesFormName';
export const PROJECT_CREATE_FORM_SELECTOR = formValueSelector(PROJECT_CREATE_FORM_NAME);

export const isProjectOrRunProcessRoute = (router: RouterState) => {
    const pathname = router.location ? router.location.pathname : '';
    const matchProject = matchProjectRoute(pathname);
    const matchRunProcess = matchRunProcessRoute(pathname);
    return Boolean(matchProject || matchRunProcess);
};

export const openProjectCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { router } = getState();
        if (!isProjectOrRunProcessRoute(router)) {
            const userUuid = getUserUuid(getState());
            if (!userUuid) { return; }
            dispatch(initialize(PROJECT_CREATE_FORM_NAME, { ownerUuid: userUuid }));
        } else {
            dispatch(initialize(PROJECT_CREATE_FORM_NAME, { ownerUuid }));
        }
        dispatch(dialogActions.OPEN_DIALOG({
            id: PROJECT_CREATE_FORM_NAME,
            data: {
                sourcePanel: GroupClass.PROJECT,
            }
        }));
    };

export const createProject = (project: Partial<ProjectResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROJECT_CREATE_FORM_NAME));
        try {
            dispatch(progressIndicatorActions.START_WORKING(PROJECT_CREATE_FORM_NAME));
            const newProject = await services.projectService.create(project, false);
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_CREATE_FORM_NAME }));
            dispatch(reset(PROJECT_CREATE_FORM_NAME));
            return newProject;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(PROJECT_CREATE_FORM_NAME, { name: 'Project with the same name already exists.' } as FormErrors));
            } else {
                dispatch(stopSubmit(PROJECT_CREATE_FORM_NAME));
                dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_CREATE_FORM_NAME }));
                const errMsg = e.errors
                    ? e.errors.join('')
                    : 'There was an error while creating the collection';
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: errMsg,
                    hideDuration: 2000,
                    kind: SnackbarKind.ERROR
                }));
            }
            return undefined;
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(PROJECT_CREATE_FORM_NAME));
        }
    };
