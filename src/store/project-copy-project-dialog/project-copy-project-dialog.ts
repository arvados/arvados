// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { dialogActions } from "~/store/dialog/dialog-actions";
import { initialize, startSubmit, stopSubmit } from 'redux-form';
import { resetPickerProjectTree } from '~/store/project-tree-picker/project-tree-picker-actions';
import { ProjectCopyFormDialogData } from "~/store/project-copy-dialog/project-copy-dialog";

export const PROJECT_COPY_DIALOG = 'projectCopy';

export const openProjectCopyDialog = (data: { projectUuid: string, name: string }) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        const initialData: ProjectCopyFormDialogData = { name: `Copy of: ${data.name}`, projectUuid: '', uuid: data.projectUuid };
        dispatch<any>(initialize(PROJECT_COPY_DIALOG, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_COPY_DIALOG, data: {} }));
    };

export const copyProject = (data: {}) =>
    (dispatch: Dispatch) => {
        dispatch(startSubmit(PROJECT_COPY_DIALOG));
        setTimeout(() => dispatch(stopSubmit(PROJECT_COPY_DIALOG, { name: 'Invalid path' })), 2000);
    };