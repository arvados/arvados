// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { addCollection, COLLECTION_CREATE_FORM_NAME, CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { UploadFile } from "~/store/collections/uploader/collection-uploader-actions";
import { DialogCollectionCreate } from "~/views-components/dialog-create/dialog-collection-create";

export const CreateCollectionDialog = compose(
    withDialog(COLLECTION_CREATE_FORM_NAME),
    reduxForm<CollectionCreateFormDialogData>({
        form: COLLECTION_CREATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            console.log('onSubmit: ', data);
            dispatch(addCollection(data));
        }
    })
)(DialogCollectionCreate);