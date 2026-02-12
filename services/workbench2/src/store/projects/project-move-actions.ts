// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "store/dialog/dialog-actions";
import { ServiceRepository } from "services/services";
import { RootState } from "store/store";
import { getUserUuid } from "common/getuser";
import { getCommonResourceServiceError, CommonResourceServiceError } from "services/common-service/common-resource-service";
import { MoveToFormDialogData } from "store/move-to-dialog/move-to-dialog";
import { resetPickerProjectTree } from "store/project-tree-picker/project-tree-picker-actions";
import { initProjectsTreePicker } from "store/tree-picker/tree-picker-actions";
import { projectPanelDataActions } from "store/project-panel/project-panel-action-bind";
import { loadSidePanelTreeProjects } from "../side-panel-tree/side-panel-tree-actions";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";

export const PROJECT_MOVE_FORM_NAME = "projectMoveFormName";

export const openMoveProjectDialog = (resource: any) => {
    return (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        dispatch<any>(initProjectsTreePicker(PROJECT_MOVE_FORM_NAME));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_MOVE_FORM_NAME, data: resource }));
    };
};

export const moveProject = (resource: MoveToFormDialogData) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const userUuid = getUserUuid(getState());
    if (!userUuid) {
        return;
    }
    try {
        const newProject = await services.projectService.update(resource.uuid, { ownerUuid: resource.ownerUuid });
        dispatch(projectPanelDataActions.REQUEST_ITEMS());

        dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_MOVE_FORM_NAME }));
        await dispatch<any>(loadSidePanelTreeProjects(userUuid));
        return newProject;
    } catch (e) {
        const error = getCommonResourceServiceError(e);
        if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "A project with the same name already exists in the target project.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        } else if (error === CommonResourceServiceError.OWNERSHIP_CYCLE) {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: "Cannot move a project into itself or one of its sub-projects.", hideDuration: 2000, kind: SnackbarKind.ERROR }));
        } else {
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_MOVE_FORM_NAME }));
            throw new Error("Could not move the project.");
        }
        return;
    }
};
