// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { ConfirmationDialog } from "../../components/confirmation-dialog/confirmation-dialog";
import { withDialog } from "../../store/dialog/with-dialog";
import { dialogActions } from "../../store/dialog/dialog-actions";
import { snackbarActions } from "../../store/snackbar/snackbar-actions";

const MULTIPLE_FILES_REMOVE_DIALOG = 'multipleFilesRemoveDialog';

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onConfirm: () => {
        // TODO: dispatch action that removes multiple files
        dispatch(dialogActions.CLOSE_DIALOG({ id: MULTIPLE_FILES_REMOVE_DIALOG }));
        dispatch(snackbarActions.OPEN_SNACKBAR({message: 'Removing files...', hideDuration: 2000}));
        setTimeout(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({message: 'Files removed.', hideDuration: 2000}));
        }, 1000);
    }
});

export const openMultipleFilesRemoveDialog = () =>
    dialogActions.OPEN_DIALOG({
        id: MULTIPLE_FILES_REMOVE_DIALOG,
        data: {
            title: 'Removing files',
            text: 'Are you sure you want to remove selected files?',
            confirmButtonLabel: 'Remove'
        }
    });

export const [MultipleFilesRemoveDialog] = [ConfirmationDialog]
    .map(withDialog(MULTIPLE_FILES_REMOVE_DIALOG))
    .map(connect(undefined, mapDispatchToProps));