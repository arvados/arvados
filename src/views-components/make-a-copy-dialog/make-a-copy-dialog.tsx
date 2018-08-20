// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from "redux";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { MakeACopyDialog, CopyFormData } from "../../components/make-a-copy/make-a-copy";
import { reduxForm, startSubmit, stopSubmit, initialize } from 'redux-form';
import { resetPickerProjectTree } from "~/store/project-tree-picker/project-tree-picker-actions";

export const MAKE_A_COPY_DIALOG = 'makeACopyDialog';
export const openMakeACopyDialog = (data: {projectUuid: string, name: string}) =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        const initialData: CopyFormData = {name: "Copy of: " + data.name, projectUuid: '', uuid: data.projectUuid};
        dispatch<any>(initialize(MAKE_A_COPY_DIALOG, initialData));
        dispatch(dialogActions.OPEN_DIALOG({ id: MAKE_A_COPY_DIALOG, data: {} }));
    };
export const MakeACopyToProjectDialog = compose(
    withDialog(MAKE_A_COPY_DIALOG),
    reduxForm({
        form: MAKE_A_COPY_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(MAKE_A_COPY_DIALOG));
            setTimeout(() => dispatch(stopSubmit(MAKE_A_COPY_DIALOG, { name: 'Invalid path' })), 2000);
        }
    })
)(MakeACopyDialog);