// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { removeRepository, REPOSITORY_REMOVE_DIALOG } from 'store/repositories/repositories-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeRepository(props.data.uuid));
    }
});

export const RemoveRepositoryDialog = compose(
    withDialog(REPOSITORY_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);