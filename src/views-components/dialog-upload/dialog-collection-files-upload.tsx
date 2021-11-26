// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from 'store/collections/collection-create-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { require } from 'validators/require';
import { FileUploaderField } from 'views-components/file-uploader/file-uploader';
import { WarningCollection } from 'components/warning-collection/warning-collection';
import { fileUploaderActions } from 'store/file-uploader/file-uploader-actions';
import { progressIndicatorActions } from 'store/progress-indicator/progress-indicator-actions';

type DialogCollectionFilesUploadProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionFilesUpload = (props: DialogCollectionFilesUploadProps) => {

    return <FormDialog
        dialogTitle='Upload data'
        formFields={UploadCollectionFilesFields}
        submitLabel='Upload data'
        doNotDisableCancel
        cancelCallback={() => {
            const { submitting, dispatch } = (props as any);

            if (submitting) {
                dispatch(progressIndicatorActions.STOP_WORKING('uploadCollectionFilesDialog'));
                dispatch(fileUploaderActions.CANCEL_FILES_UPLOAD());
                dispatch(fileUploaderActions.CLEAR_UPLOAD());
            }
        }}
        {...props}
    />;
}

const UploadCollectionFilesFields = () => <>
    <Field
        name='files'
        validate={FILES_FIELD_VALIDATION}
        component={FileUploaderField} />
    <WarningCollection text="Uploading new files will change content address." />
</>;

const FILES_FIELD_VALIDATION = [require];


