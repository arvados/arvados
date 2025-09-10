// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogExternalCredentialCreate } from "./dialog-external-credential-create";
import { NEW_EXTERNAL_CREDENTIAL_FORM_NAME } from "store/external-credentials/external-credentials-actions";
import { ExternalCredentialCreateFormDialogData } from "store/external-credentials/external-credential-dialog-data";
import { createExternalCredential } from "store/external-credentials/external-credentials-actions";

export const CreateExternalCredentialDialog = compose(
    withDialog(NEW_EXTERNAL_CREDENTIAL_FORM_NAME),
    reduxForm<ExternalCredentialCreateFormDialogData>({
        form: NEW_EXTERNAL_CREDENTIAL_FORM_NAME,
        onSubmit: (data, dispatch) => {
            if (data.name) {
                data.name = data.name.trim();
            }
            dispatch(createExternalCredential(data));
            return;
        }
    })
)(DialogExternalCredentialCreate);