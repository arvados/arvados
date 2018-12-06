// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reset, startSubmit, stopSubmit, initialize, FormErrors } from 'redux-form';
import { RootState } from '~/store/store';
import { dialogActions } from "~/store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { ProjectResource } from '~/models/project';
import { ServiceRepository } from '~/services/services';
import { matchProjectRoute, matchRunProcessRoute } from '~/routes/routes';

export interface ProjectCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
}

export const PROJECT_CREATE_FORM_NAME = 'projectCreateFormName';

export const isProjectOrRunProcessRoute = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const matchProject = matchProjectRoute(pathname);
    const matchRunProcess = matchRunProcessRoute(pathname);
    return Boolean(matchProject || matchRunProcess);
};

export const isItemNotInProject = (properties: any) => {
    if (properties.breadcrumbs) {
        return Boolean(properties.breadcrumbs[0].label !== 'Projects');
    } else {
        return ;
    }
};

export const openProjectCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const router = getState();
        const properties = getState().properties;
        if (isItemNotInProject(properties) || !isProjectOrRunProcessRoute(router)) {
            const userUuid = getState().auth.user!.uuid;
            dispatch(initialize(PROJECT_CREATE_FORM_NAME, { userUuid }));
        } else {
            dispatch(initialize(PROJECT_CREATE_FORM_NAME, { ownerUuid }));
        }
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_CREATE_FORM_NAME, data: {} }));
    };

export const createProject = (project: Partial<ProjectResource>) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(startSubmit(PROJECT_CREATE_FORM_NAME));
        try {
            const newProject = await services.projectService.create(project);
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_CREATE_FORM_NAME }));
            dispatch(reset(PROJECT_CREATE_FORM_NAME));
            return newProject;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROJECT_CREATE_FORM_NAME, { name: 'Project with the same name already exists.' } as FormErrors));
            }
            return undefined;
        }
    };
