// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { CollectionPartialCopyToExistingCollectionFormData, copyCollectionPartialToExistingCollection, COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION } from 'store/collections/collection-partial-copy-actions';
import { DialogCollectionPartialCopyToExistingCollection } from "views-components/dialog-copy/dialog-collection-partial-copy-to-existing-collection";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialCopyToExistingCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION),
    reduxForm<CollectionPartialCopyToExistingCollectionFormData>({
        form: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollectionPartialToExistingCollection(data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION),
)(DialogCollectionPartialCopyToExistingCollection);
