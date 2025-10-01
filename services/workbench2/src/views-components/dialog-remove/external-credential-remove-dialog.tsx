// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { removeExternalCredentialPermanently, REMOVE_EXTERNAL_CREDENTIAL_DIALOG } from 'store/external-credentials/external-credentials-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeExternalCredentialPermanently(props.data.uuid));
    }
});

export const RemoveExternalCredentialDialog = compose(
    withDialog(REMOVE_EXTERNAL_CREDENTIAL_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);
