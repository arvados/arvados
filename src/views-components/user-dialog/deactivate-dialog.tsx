// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { deactivate, DEACTIVATE_DIALOG } from 'store/user-profile/user-profile-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(deactivate(props.data.uuid));
    }
});

export const DeactivateDialog = compose(
    withDialog(DEACTIVATE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
