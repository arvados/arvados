// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS, CollectionPartialCopyToSeparateCollectionsFormData, copyCollectionPartialToSeparateCollections } from 'store/collections/collection-partial-copy-actions';
import { DialogCollectionPartialCopyToSeparateCollection } from "views-components/dialog-copy/dialog-collection-partial-copy-to-separate-collections";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialCopyToSeparateCollectionsDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS),
    reduxForm<CollectionPartialCopyToSeparateCollectionsFormData>({
        form: COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollectionPartialToSeparateCollections(data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS),
)(DialogCollectionPartialCopyToSeparateCollection);
