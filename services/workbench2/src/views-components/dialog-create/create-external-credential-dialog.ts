// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogExternalCredentialCreate } from "./dialog-external-credential-create";
import { CREATE_EXTERNAL_CREDENTIAL_FORM_NAME } from "store/external-credentials/external-credentials-actions";
import { CreateExternalCredentialFormDialogData } from "store/external-credentials/external-credential-dialog-data";
import { createExternalCredential } from "store/external-credentials/external-credentials-actions";

export const CreateExternalCredentialDialog = compose(
    withDialog(CREATE_EXTERNAL_CREDENTIAL_FORM_NAME),
    reduxForm<CreateExternalCredentialFormDialogData>({
        form: CREATE_EXTERNAL_CREDENTIAL_FORM_NAME,
        onSubmit: (data: CreateExternalCredentialFormDialogData, dispatch: Dispatch) => {
            for (const key in data) {
                if (typeof data[key] === 'string') {
                    data[key] = data[key].trim();
                }
            }
            dispatch<any>(createExternalCredential(data));
            return;
        }
    })
)(DialogExternalCredentialCreate);