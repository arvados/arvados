// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { setup, SETUP_DIALOG } from 'store/user-profile/user-profile-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(setup(props.data.uuid));
    }
});

export const SetupDialog = compose(
    withDialog(SETUP_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
