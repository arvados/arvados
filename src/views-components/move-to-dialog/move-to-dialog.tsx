// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from "redux";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { MoveToDialog } from "../../components/move-to-dialog/move-to-dialog";
import { reduxForm, startSubmit, stopSubmit } from "redux-form";
import { resetPickerProjectTree } from "~/store/project-tree-picker/project-tree-picker-actions";

export const MOVE_TO_DIALOG = 'moveToDialog';

export const openMoveToDialog = () =>
    (dispatch: Dispatch) => {
        dispatch<any>(resetPickerProjectTree());
        dispatch(dialogActions.OPEN_DIALOG({ id: MOVE_TO_DIALOG, data: {} }));
    };

export const MoveToProjectDialog = compose(
    withDialog(MOVE_TO_DIALOG),
    reduxForm({
        form: MOVE_TO_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(MOVE_TO_DIALOG));
            setTimeout(() => dispatch(stopSubmit(MOVE_TO_DIALOG, { name: 'Invalid path' })), 2000);
        }
    })
)(MoveToDialog);
