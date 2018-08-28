// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { COLLECTION_UPLOAD_FILES_DIALOG, submitCollectionFiles } from '~/store/collections/collection-upload-actions';
import { CollectionFilesUploadDialog as Dialog } from '../dialog-upload/collection-files-upload-dialog';

export const CollectionFilesUploadDialog = compose(
    withDialog(COLLECTION_UPLOAD_FILES_DIALOG),
    reduxForm<CollectionCreateFormDialogData>({
        form: COLLECTION_UPLOAD_FILES_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(submitCollectionFiles());
        }
    })
)(Dialog);
