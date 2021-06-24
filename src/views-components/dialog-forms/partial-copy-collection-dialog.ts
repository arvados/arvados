// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from 'store/dialog/with-dialog';
import { CollectionPartialCopyFormData, copyCollectionPartial, COLLECTION_PARTIAL_COPY_FORM_NAME } from 'store/collections/collection-partial-copy-actions';
import { DialogCollectionPartialCopy } from "views-components/dialog-copy/dialog-collection-partial-copy";
import { pickerId } from "store/tree-picker/picker-id";


export const PartialCopyCollectionDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_FORM_NAME),
    reduxForm<CollectionPartialCopyFormData>({
        form: COLLECTION_PARTIAL_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollectionPartial(data));
        }
    }),
    pickerId(COLLECTION_PARTIAL_COPY_FORM_NAME),
)(DialogCollectionPartialCopy);