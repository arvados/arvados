// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { reduxForm, reset, startSubmit, stopSubmit } from "redux-form";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { RenameDialog } from "../../components/rename-dialog/rename-dialog";

export const RENAME_FILE_DIALOG = 'renameFileDialog';

export const openRenameFileDialog = (originalName: string) =>
    (dispatch: Dispatch) => {
        dispatch(reset(RENAME_FILE_DIALOG));
        dispatch(dialogActions.OPEN_DIALOG({ id: RENAME_FILE_DIALOG, data: originalName }));
    };

export const [RenameFileDialog] = [RenameDialog]
    .map(withDialog(RENAME_FILE_DIALOG))
    .map(reduxForm({
        form: RENAME_FILE_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(startSubmit(RENAME_FILE_DIALOG));
            // TODO: call collection file renaming action here
            setTimeout(() => dispatch(stopSubmit(RENAME_FILE_DIALOG, { name: 'Invalid name' })), 2000);
        }
    }));
