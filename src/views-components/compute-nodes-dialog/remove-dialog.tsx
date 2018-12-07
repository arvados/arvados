// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "~/components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { COMPUTE_NODE_REMOVE_DIALOG, removeComputeNode } from '~/store/compute-nodes/compute-nodes-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeComputeNode(props.data.uuid));
    }
});

export const  RemoveComputeNodeDialog = compose(
    withDialog(COMPUTE_NODE_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);