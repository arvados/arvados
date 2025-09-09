// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogExternalCredentialUpdate } from "views-components/dialog-update/dialog-external-credential-update";
import { ExternalCredentialUpdateFormDialogData } from "store/external-credentials/external-credential-dialog-data";
import { EXTERNAL_CREDENTIAL_UPDATE_FORM_NAME } from "store/external-credentials/external-credentials-actions";
import { updateExternalCredential } from "store/external-credentials/external-credentials-actions";

export const UpdateExternalCredentialDialog = compose(
    withDialog(EXTERNAL_CREDENTIAL_UPDATE_FORM_NAME),
    reduxForm<ExternalCredentialUpdateFormDialogData>({
        form: EXTERNAL_CREDENTIAL_UPDATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            if (data.scopes && typeof data.scopes === 'string') {
                data.scopes = data.scopes.split(',').reduce((acc: string[], s: string) => {
                    const trimmed = s.trim();
                    if (trimmed) {
                        acc.push(trimmed);
                    }
                    return acc;
                }, []);
            }
            dispatch(updateExternalCredential(data));
        }
    })
)(DialogExternalCredentialUpdate);
