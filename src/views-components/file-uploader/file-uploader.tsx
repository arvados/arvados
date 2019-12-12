// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { FileUpload } from '~/components/file-upload/file-upload';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { FileUploadProps } from '../../components/file-upload/file-upload';
import { Dispatch } from 'redux';
import { fileUploaderActions, getFileUploaderState } from '~/store/file-uploader/file-uploader-actions';
import { WrappedFieldProps } from 'redux-form';
import { Typography } from '@material-ui/core';

export type FileUploaderProps = Pick<FileUploadProps, 'disabled' | 'onDrop'>;

const mapStateToProps = (state: RootState, { disabled }: FileUploaderProps): Pick<FileUploadProps, 'files' | 'disabled'> => ({
    disabled,
    files: state.fileUploader,
});

const mapDispatchToProps = (dispatch: Dispatch, { onDrop }: FileUploaderProps): Pick<FileUploadProps, 'onDrop' | 'onDelete'> => ({
    onDrop: files => {
        const state = dispatch<any>(getFileUploaderState());
        if (files.length > 0 && state.length === 0) {
            dispatch(fileUploaderActions.SET_UPLOAD_FILES(files));
            onDrop(files);
        } else if (files.length > 0 && state.length > 0) {
            dispatch(fileUploaderActions.UPDATE_UPLOAD_FILES(files));
            onDrop(files);
        }
    },
    onDelete: file => dispatch(fileUploaderActions.DELETE_UPLOAD_FILE(file)),
});

export const FileUploader = connect(mapStateToProps, mapDispatchToProps)(FileUpload);

export const FileUploaderField = (props: WrappedFieldProps & { label?: string }) =>
    <div>
        <Typography variant='caption'>{props.label}</Typography>
        <FileUploader disabled={props.meta.submitting} onDrop={props.input.onChange} />
    </div>;
