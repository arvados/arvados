// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { KEEP_SERVICE_REMOVE_DIALOG, removeKeepService } from 'store/keep-services/keep-services-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeKeepService(props.data.uuid));
    }
});

export const RemoveKeepServiceDialog = compose(
    withDialog(KEEP_SERVICE_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);