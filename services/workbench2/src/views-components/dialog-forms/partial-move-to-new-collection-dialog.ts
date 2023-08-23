// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION, CollectionPartialMoveToNewCollectionFormData, moveCollectionPartialToNewCollection } from "store/collections/collection-partial-move-actions";
import { DialogCollectionPartialMoveToNewCollection } from "views-components/dialog-move/dialog-collection-partial-move-to-new-collection";
import { pickerId } from "store/tree-picker/picker-id";

export const PartialMoveToNewCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION),
    reduxForm<CollectionPartialMoveToNewCollectionFormData>({
        form: COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION,
        onSubmit: (data, dispatch, dialog) => {
            dispatch(moveCollectionPartialToNewCollection(dialog.data, data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION),
)(DialogCollectionPartialMoveToNewCollection);
