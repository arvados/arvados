// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "~/components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { VIRTUAL_MACHINE_REMOVE_DIALOG, removeVirtualMachine } from '~/store/virtual-machines/virtual-machines-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeVirtualMachine(props.data.uuid));
    }
});

export const RemoveVirtualMachineDialog = compose(
    withDialog(VIRTUAL_MACHINE_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);