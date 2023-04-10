// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION, CollectionPartialMoveToExistingCollectionFormData, moveCollectionPartialToExistingCollection } from "store/collections/collection-partial-move-actions";
import { DialogCollectionPartialMoveToExistingCollection } from "views-components/dialog-move/dialog-collection-partial-move-to-existing-collection";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialMoveToExistingCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION),
    reduxForm<CollectionPartialMoveToExistingCollectionFormData>({
        form: COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION,
        onSubmit: (data, dispatch) => {
            dispatch(moveCollectionPartialToExistingCollection(data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION),
)(DialogCollectionPartialMoveToExistingCollection);
