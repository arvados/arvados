// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose, Dispatch } from "redux";
import { connect } from 'react-redux';
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { DialogFileUploaderField } from 'views-components/file-uploader/file-uploader';
import { WarningCollection } from 'components/warning-collection/warning-collection';
import { DialogTitle, DialogContent } from '@mui/material';
import { COLLECTION_UPLOAD_FILES_DIALOG, submitCollectionFiles } from 'store/collections/collection-upload-actions';

type DialogCollectionFilesUploadProps = WithDialogProps<{targetLocation?: string}> & {
    submitCollectionFiles: (targetLocation?: string) => void;
};

const mapDispatch = (dispatch: Dispatch) => ({
    submitCollectionFiles: (targetLocation?: string) => dispatch<any>(submitCollectionFiles(targetLocation))
});

export const DialogCollectionFilesUpload = compose(
    connect(null, mapDispatch),
    withDialog(COLLECTION_UPLOAD_FILES_DIALOG)
)((props: DialogCollectionFilesUploadProps) => {
    const { open, data, closeDialog } = props;
    const [isPopulated, setIsPopulated] = React.useState(false);
    const [isSubmitting, setIsSubmitting] = React.useState(false);

    const fields = () => (
        <>
            <DialogTitle>Upload data</DialogTitle>
            <DialogContent>
                <DialogFileUploaderField onDrop={(files: File[]) => setIsPopulated(files.length > 0)} />
                <WarningCollection text="Uploading new files will change content address. Empty folders will be ignored." />
            </DialogContent>
        </>
    );

    return (
        <DialogForm
            open={open}
            fields={fields()}
            submitLabel="Upload data"
            formErrors={isPopulated ? [] : ['Please add files to upload']} // content of err string doesn't matter here
            isSubmitting={isSubmitting}
            onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
                event.preventDefault();
                setIsSubmitting(true);
                props.submitCollectionFiles(data?.targetLocation);
            }}
            closeDialog={closeDialog}
            clearFormValues={() => {
                setIsPopulated(false);
                setIsSubmitting(false);
            }}
        />
    );
});
