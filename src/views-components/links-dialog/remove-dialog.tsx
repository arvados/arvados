// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { LINK_REMOVE_DIALOG, removeLink } from 'store/link-panel/link-panel-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeLink(props.data.uuid));
    }
});

export const RemoveLinkDialog = compose(
    withDialog(LINK_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);