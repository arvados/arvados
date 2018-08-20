// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch, compose } from "redux";
import { withDialog } from '~/store/dialog/with-dialog';
import { FilesUploadDialog } from '~/components/file-upload-dialog/file-upload-dialog';
import { RootState } from '../../store/store';
import { uploadCurrentCollectionFiles, UPLOAD_COLLECTION_FILES_DIALOG, collectionUploaderActions } from '~/store/collections/uploader/collection-uploader-actions';

const mapStateToProps = (state: RootState) => ({
    files: state.collections.uploader,
    uploading: state.collections.uploader.some(file => file.loaded < file.total)
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onSubmit: () => {
        dispatch<any>(uploadCurrentCollectionFiles());
    },
    onChange: (files: File[]) => {
        dispatch(collectionUploaderActions.SET_UPLOAD_FILES(files));
    }
});

export const UploadCollectionFilesDialog = compose(
    withDialog(UPLOAD_COLLECTION_FILES_DIALOG),
    connect(mapStateToProps, mapDispatchToProps)
)(FilesUploadDialog);