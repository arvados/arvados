// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { COLLECTION_CREATE_FORM_NAME, CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { DialogCollectionCreate } from "~/views-components/dialog-create/dialog-collection-create";
import { createCollection } from "~/store/workbench/workbench-actions";

export const CreateCollectionDialog = compose(
    withDialog(COLLECTION_CREATE_FORM_NAME),
    reduxForm<CollectionCreateFormDialogData>({
        form: COLLECTION_CREATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(createCollection(data));
        }
    })
)(DialogCollectionCreate);

// onSubmit: (data: { name: string, description: string }, files: UploadFile[]) => void;