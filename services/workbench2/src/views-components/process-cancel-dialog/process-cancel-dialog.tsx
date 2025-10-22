// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { CANCEL_PROCESS_DIALOG, cancelRunningWorkflow } from 'store/processes/processes-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(cancelRunningWorkflow(props.data.uuid));
    }
});

export const CancelProcessDialog = compose(
    withDialog(CANCEL_PROCESS_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
