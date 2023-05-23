// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS, CollectionPartialMoveToSeparateCollectionsFormData, moveCollectionPartialToSeparateCollections } from "store/collections/collection-partial-move-actions";
import { DialogCollectionPartialMoveToSeparateCollections } from "views-components/dialog-move/dialog-collection-partial-move-to-separate-collections";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialMoveToSeparateCollectionsDialog = compose(
    withDialog(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS),
    reduxForm<CollectionPartialMoveToSeparateCollectionsFormData>({
        form: COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS,
        onSubmit: (data, dispatch, dialog) => {
            dispatch(moveCollectionPartialToSeparateCollections(dialog.data, data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS),
)(DialogCollectionPartialMoveToSeparateCollections);
