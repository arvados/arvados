// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { ConfirmationDialog } from "../../components/confirmation-dialog/confirmation-dialog";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { snackbarActions } from "../../store/snackbar/snackbar-actions";

const FILE_REMOVE_DIALOG = 'fileRemoveDialog';

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onConfirm: () => {
        // TODO: dispatch action that removes single file
        dispatch(dialogActions.CLOSE_DIALOG({ id: FILE_REMOVE_DIALOG }));
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing file...', hideDuration: 2000 }));
        setTimeout(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'File removed.', hideDuration: 2000 }));
        }, 1000);
    }
});

export const openFileRemoveDialog = (fileId: string) =>
    dialogActions.OPEN_DIALOG({
        id: FILE_REMOVE_DIALOG,
        data: {
            title: 'Removing file',
            text: 'Are you sure you want to remove this file?',
            confirmButtonLabel: 'Remove',
            fileId
        }
    });

export const [FileRemoveDialog] = [ConfirmationDialog]
    .map(withDialog(FILE_REMOVE_DIALOG))
    .map(connect(undefined, mapDispatchToProps));