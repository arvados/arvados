// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from 'store/store';
import { setWorkflow, SET_WORKFLOW_DIALOG } from 'store/run-process-panel/run-process-panel-actions';
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { WorkflowResource } from 'models/workflow';

const mapStateToProps = (state: RootState, props: WithDialogProps<{ workflow: WorkflowResource }>) => ({
    workflow: props.data.workflow
});

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: (workflow: WorkflowResource) => {
        props.closeDialog();
        dispatch<any>(setWorkflow(workflow));
    }
});

const mergeProps = (
    stateProps: { workflow: WorkflowResource },
    dispatchProps: { onConfirm: (workflow: WorkflowResource) => void },
    props: WithDialogProps<{ workflow: WorkflowResource }>) => ({
        onConfirm: () => dispatchProps.onConfirm(stateProps.workflow),
        ...props
    });

export const [ChangeWorkflowDialog] = [ConfirmationDialog]
    .map(connect(mapStateToProps, mapDispatchToProps, mergeProps) as any)
    .map(withDialog(SET_WORKFLOW_DIALOG));