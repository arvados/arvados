// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { FileUpload } from "~/components/file-upload/file-upload";
import { Dialog, DialogTitle, DialogContent, DialogActions } from '@material-ui/core/';
import { Button, CircularProgress } from '@material-ui/core';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { UploadFile } from '~/store/file-uploader/file-uploader-actions';

export interface FilesUploadDialogProps {
    files: UploadFile[];
    uploading: boolean;
    onSubmit: () => void;
    onChange: (files: File[]) => void;
}

export const FilesUploadDialog = (props: FilesUploadDialogProps & WithDialogProps<{}>) =>
    <Dialog open={props.open}
        disableBackdropClick={true}
        disableEscapeKeyDown={true}
        fullWidth={true}
        maxWidth='sm'>
        <DialogTitle>Upload data</DialogTitle>
        <DialogContent>
            <FileUpload
                files={props.files}
                disabled={props.uploading}
                onDrop={props.onChange}
            />
        </DialogContent>
        <DialogActions>
            <Button
                variant='text'
                color='primary'
                disabled={props.uploading}
                onClick={props.closeDialog}>
                Cancel
            </Button>
            <Button
                variant='contained'
                color='primary'
                type='submit'
                onClick={props.onSubmit}
                disabled={props.uploading}>
                {props.uploading
                    ? <CircularProgress size={20} />
                    : 'Upload data'}
            </Button>
        </DialogActions>
    </Dialog>;
