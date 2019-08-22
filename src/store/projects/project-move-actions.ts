// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { startSubmit, stopSubmit, initialize, FormErrors } from 'redux-form';
import { ServiceRepository } from '~/services/services';
import { RootState } from '~/store/store';
import { getCommonResourceServiceError, CommonResourceServiceError } from "~/services/common-service/common-resource-service";
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { initProjectsTreePicker } from '~/store/tree-picker/tree-picker-actions';
import { projectPanelActions } from '~/store/project-panel/project-panel-action';
import { loadSidePanelTreeProjects } from '../side-panel-tree/side-panel-tree-actions';

export const PROJECT_MOVE_FORM_NAME = 'projectMoveFormName';

export const openMoveProjectDialog = (resource: { name: string, uuid: string }) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        dispatch<any>(initProjectsTreePicker(PROJECT_MOVE_FORM_NAME));
        dispatch(initialize(PROJECT_MOVE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_MOVE_FORM_NAME, data: {} }));
    };

export const moveProject = (resource: MoveToFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const userUuid = getState().auth.user!.uuid;
        dispatch(startSubmit(PROJECT_MOVE_FORM_NAME));
        try {
            const newProject = await services.projectService.update(resource.uuid, { ownerUuid: resource.ownerUuid });
            dispatch(projectPanelActions.REQUEST_ITEMS());
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_MOVE_FORM_NAME }));
            await dispatch<any>(loadSidePanelTreeProjects(userUuid));
            return newProject;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(PROJECT_MOVE_FORM_NAME, { ownerUuid: 'A project with the same name already exists in the target project.' } as FormErrors));
            } else if (error === CommonResourceServiceError.OWNERSHIP_CYCLE) {
                dispatch(stopSubmit(PROJECT_MOVE_FORM_NAME, { ownerUuid: 'Cannot move a project into itself.' } as FormErrors));
            } else {
                dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_MOVE_FORM_NAME }));
                throw new Error('Could not move the project.');
            }
            return;
        }
    };
