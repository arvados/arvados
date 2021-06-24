// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';
import { RootState } from 'store/store';
import { removeCollectionFiles, FILE_REMOVE_DIALOG } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';

const mapStateToProps = (state: RootState, props: WithDialogProps<{ filePath: string }>) => ({
    filePath: props.data.filePath
});

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<{ filePath: string }>) => ({
    onConfirm: (filePath: string) => {
        props.closeDialog();
        dispatch<any>(removeCollectionFiles([filePath]));
    }
});

const mergeProps = (
    stateProps: { filePath: string },
    dispatchProps: { onConfirm: (filePath: string) => void },
    props: WithDialogProps<{ filePath: string }>) => ({
        onConfirm: () => dispatchProps.onConfirm(stateProps.filePath),
        ...props
    });

// TODO: Remove as any
export const [FileRemoveDialog] = [ConfirmationDialog]
    .map(connect(mapStateToProps, mapDispatchToProps, mergeProps) as any)
    .map(withDialog(FILE_REMOVE_DIALOG));
