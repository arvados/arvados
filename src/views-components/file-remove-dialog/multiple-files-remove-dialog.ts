// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { ConfirmationDialog } from "../../components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "../../store/dialog/with-dialog";
import { MULTIPLE_FILES_REMOVE_DIALOG, removeCollectionsSelectedFiles } from "../../store/collection-panel/collection-panel-files/collection-panel-files-actions";

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeCollectionsSelectedFiles());
    }
});

export const [MultipleFilesRemoveDialog] = [ConfirmationDialog]
    .map(connect(undefined, mapDispatchToProps))
    .map(withDialog(MULTIPLE_FILES_REMOVE_DIALOG));