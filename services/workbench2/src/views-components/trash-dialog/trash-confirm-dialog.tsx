// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { toggleResourceTrashed, TRASH_CONFIRM_DIALOG } from 'store/trash/trash-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(toggleResourceTrashed(props.data.uuids, props.data.isTrashed));
    }
});

export const TrashConfirmDialog = compose(
    withDialog(TRASH_CONFIRM_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
