// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch, compose } from 'redux';
import { connect } from "react-redux";
import { ConfirmationDialog } from "~/components/confirmation-dialog/confirmation-dialog";
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { COLLECTION_RECOVER_VERSION_DIALOG, recoverVersion } from '~/store/collections/collection-version-actions';

const mapDispatchToProps = (dispatch: Dispatch, props: WithDialogProps<any>) => ({
    onConfirm: () => {
        props.closeDialog();
        dispatch<any>(recoverVersion(props.data.uuid));
    }
});

export const RecoverCollectionVersionDialog = compose(
    withDialog(COLLECTION_RECOVER_VERSION_DIALOG),
    connect(null, mapDispatchToProps)
)(ConfirmationDialog);