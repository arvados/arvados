// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose, Dispatch } from 'redux';
import { reduxForm, InjectedFormProps, Field } from 'redux-form';
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { DialogContentText } from '@mui/material';
import { TextField } from 'components/text-field/text-field';
import { DOWNLOAD_ZIP_DIALOG, downloadZip } from 'store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { DOWNLOAD_ZIP_VALIDATION } from 'validators/validators';

interface DownloadZipFormData {
    collectionUuid: string;
    fileName: string;
    paths: string[];
}

export const DownloadFilesAsZipDialog = compose(
    withDialog(DOWNLOAD_ZIP_DIALOG),
    reduxForm({
        form: DOWNLOAD_ZIP_DIALOG,
        touchOnChange: true,
        onSubmit: (data: DownloadZipFormData, dispatch: Dispatch) => {
            dispatch<any>(downloadZip(data.collectionUuid, data.paths, data.fileName));
        }
    })
)((props: WithDialogProps<{}> & InjectedFormProps<DownloadZipFormData>) =>
    <FormDialog
        dialogTitle='Download'
        formFields={DownloadFilesAsZipFormFields}
        submitLabel='Ok'
        enableWhenPristine={true}
        {...props}
    />);

const DownloadFilesAsZipFormFields = () => <>
    <DialogContentText>
        {"Please enter a name for the downloaded zip"}
    </DialogContentText>
    <Field
        name='fileName'
        component={TextField as any}
        autoFocus={true}
        validate={DOWNLOAD_ZIP_VALIDATION}
    />
</>;
