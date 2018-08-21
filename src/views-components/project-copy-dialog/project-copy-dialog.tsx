// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from "redux";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { ProjectCopy, CopyFormData } from "../../components/project-copy/project-copy";
import { reduxForm, startSubmit, stopSubmit, initialize } from 'redux-form';
import { resetPickerProjectTree } from "~/store/project-tree-picker/project-tree-picker-actions";

export const PROJECT_COPY_DIALOG = 'projectCopy';
export const openProjectCopyDialog = (data: { projectUuid: string, name: string }) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        const initialData: CopyFormData = { name: `Copy of: ${data.name}`, projectUuid: '', uuid: data.projectUuid };
        dispatch<any>(initialize(PROJECT_COPY_DIALOG, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_COPY_DIALOG, data: {} }));
    };

export const ProjectCopyDialog = compose(
    withDialog(PROJECT_COPY_DIALOG),
    reduxForm({
        form: PROJECT_COPY_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(PROJECT_COPY_DIALOG));
            setTimeout(() => dispatch(stopSubmit(PROJECT_COPY_DIALOG, { name: 'Invalid path' })), 2000);
        }
    })
)(ProjectCopy);