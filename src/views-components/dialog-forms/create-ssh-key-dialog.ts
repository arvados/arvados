// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import {
    SSH_KEY_CREATE_FORM_NAME,
    createSshKey,
    SshKeyCreateFormDialogData
} from 'store/auth/auth-action-ssh';
import { DialogSshKeyCreate } from 'views-components/dialog-create/dialog-ssh-key-create';

export const CreateSshKeyDialog = compose(
    withDialog(SSH_KEY_CREATE_FORM_NAME),
    reduxForm<SshKeyCreateFormDialogData>({
        form: SSH_KEY_CREATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(createSshKey(data));
        }
    })
)(DialogSshKeyCreate);
