// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { CollectionPartialCopyToSelectedCollectionFormData, copyCollectionPartialToSelectedCollection, COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION } from 'store/collections/collection-partial-copy-actions';
import { DialogCollectionPartialCopyToSelectedCollection } from "views-components/dialog-copy/dialog-partial-copy-to-collection";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialCopyToCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION),
    reduxForm<CollectionPartialCopyToSelectedCollectionFormData>({
        form: COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollectionPartialToSelectedCollection(data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION),
)(DialogCollectionPartialCopyToSelectedCollection);