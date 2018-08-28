// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog, } from '~/store/dialog/with-dialog';
import { CollectionPartialCopyFormData, doCollectionPartialCopy, COLLECTION_PARTIAL_COPY_FORM_NAME } from '~/store/collections/collection-partial-copy-actions';
import { CollectionPartialCopyDialog as Dialog } from "~/views-components/dialog-copy/collection-partial-copy-dialog";


export const CollectionPartialCopyDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_FORM_NAME),
    reduxForm({
        form: COLLECTION_PARTIAL_COPY_FORM_NAME,
        onSubmit: (data: CollectionPartialCopyFormData, dispatch) => {
            dispatch(doCollectionPartialCopy(data));
        }
    }))(Dialog);