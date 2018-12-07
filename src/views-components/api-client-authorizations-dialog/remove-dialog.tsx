// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "~/components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { API_CLIENT_AUTHORIZATION_REMOVE_DIALOG, removeApiClientAuthorization } from '~/store/api-client-authorizations/api-client-authorizations-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(removeApiClientAuthorization(props.data.uuid));
    }
});

export const RemoveApiClientAuthorizationDialog = compose(
    withDialog(API_CLIENT_AUTHORIZATION_REMOVE_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);