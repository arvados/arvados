// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { removeWorkflowPermanently, REMOVE_WORKFLOW_DIALOG } from 'store/workflow-panel/workflow-panel-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeWorkflowPermanently(props.data.uuid));
    }
});

export const RemoveWorkflowDialog = compose(
    withDialog(REMOVE_WORKFLOW_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
