// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { SSH_KEY_REMOVE_DIALOG, removeSshKey } from 'store/auth/auth-action-ssh';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeSshKey(props.data.uuid));
    }
});

export const RemoveSshKeyDialog = compose(
    withDialog(SSH_KEY_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
