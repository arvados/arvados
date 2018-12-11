// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "~/components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { removeGroup, GROUP_REMOVE_DIALOG } from '~/store/groups-panel/groups-panel-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeGroup(props.data.uuid));
    }
});

export const RemoveGroupDialog = compose(
    withDialog(GROUP_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);