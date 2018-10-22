// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { require } from '~/validators/require';
import { FileUploaderField } from '~/views-components/file-uploader/file-uploader';
import { WarningIcon } from '~/components/icon/icon';
import { DialogContentText } from '@material-ui/core';


type DialogCollectionFilesUploadProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionFilesUpload = (props: DialogCollectionFilesUploadProps) =>
    <FormDialog
        dialogTitle='Upload data'
        formFields={UploadCollectionFilesFields}
        submitLabel='Upload data'
        {...props}
    />;

const UploadCollectionFilesFields = () => <>
    <Field
        name='files'
        validate={FILES_FIELD_VALIDATION}
        component={FileUploaderField} />
    <span style={{ display: 'flex', alignItems: 'center', marginTop: '8px' }}>
        <WarningIcon />
        <DialogContentText style={{ paddingLeft: '8px' }}>Uploading new files will change content address.</DialogContentText>
    </span>
</>;


const FILES_FIELD_VALIDATION = [require];


