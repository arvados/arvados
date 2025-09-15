// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogExternalCredentialUpdate } from "views-components/dialog-update/dialog-external-credential-update";
import { UpdateExternalCredentialFormDialogData } from "store/external-credentials/external-credential-dialog-data";
import { UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME } from "store/external-credentials/external-credentials-actions";
import { updateExternalCredential } from "store/external-credentials/external-credentials-actions";

export const UpdateExternalCredentialDialog = compose(
    withDialog(UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME),
    reduxForm<UpdateExternalCredentialFormDialogData>({
        form: UPDATE_EXTERNAL_CREDENTIAL_FORM_NAME,
        onSubmit: (data: UpdateExternalCredentialFormDialogData, dispatch: Dispatch) => {
            Object.values(data).forEach(value => {
                if (value && typeof value === 'string') {
                    value = value.trim();
                }
            });
            dispatch<any>(updateExternalCredential(data));
        }
    })
)(DialogExternalCredentialUpdate);
