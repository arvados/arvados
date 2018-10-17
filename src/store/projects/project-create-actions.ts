// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reset, startSubmit, stopSubmit, initialize } from 'redux-form';
import { RootState } from '~/store/store';
import { dialogActions } from "~/store/dialog/dialog-actions";
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { ProjectResource } from '~/models/project';
import { ServiceRepository } from '~/services/services';
import { matchProjectRoute } from '~/routes/routes';

export interface ProjectCreateFormDialogData {
    ownerUuid: string;
    name: string;
    description: string;
}

export const PROJECT_CREATE_FORM_NAME = 'projectCreateFormName';

export const isProjectRoute = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchProjectRoute(pathname);
    return !!match;
};

interface Properties {
    breadcrumbs: Array<{ uuid: string, label: string }>;
}

export const isItemNotInProject = (properties: Properties) => {
    if (properties.breadcrumbs) {
        const isItemSharedWithMe = properties.breadcrumbs[0].label !== 'Projects';
        return isItemSharedWithMe;
    } else {
        return false;
    }
};

export const openProjectCreateDialog = (ownerUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        if (isItemNotInProject || !isProjectRoute) {
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
                dispatch(stopSubmit(PROJECT_CREATE_FORM_NAME, { name: 'Project with the same name already exists.' }));
            }
            return undefined;
        }
    };
