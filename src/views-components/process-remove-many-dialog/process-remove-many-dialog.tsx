// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from 'react-redux';
import { ConfirmationDialog } from 'components/confirmation-dialog/confirmation-dialog';
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';
import { removeProcessPermanently, REMOVE_MANY_PROCESSES_DIALOG } from 'store/processes/processes-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        props.data.list.forEach((uuid: string) => dispatch<any>(removeProcessPermanently(uuid)));
    },
});

export const RemoveManyProcessesDialog = compose(withDialog(REMOVE_MANY_PROCESSES_DIALOG), connect(null, mapDispatchToProps))(ConfirmationDialog);
