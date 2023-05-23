// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { CollectionPartialCopyToNewCollectionFormData, copyCollectionPartialToNewCollection, COLLECTION_PARTIAL_COPY_FORM_NAME } from 'store/collections/collection-partial-copy-actions';
import { DialogCollectionPartialCopyToNewCollection } from "views-components/dialog-copy/dialog-collection-partial-copy-to-new-collection";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialCopyToNewCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_FORM_NAME),
    reduxForm<CollectionPartialCopyToNewCollectionFormData>({
        form: COLLECTION_PARTIAL_COPY_FORM_NAME,
        onSubmit: (data, dispatch, dialog) => {
            console.log(dialog.data);
            dispatch(copyCollectionPartialToNewCollection(dialog.data, data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_COPY_FORM_NAME),
)(DialogCollectionPartialCopyToNewCollection);
