// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from '~/store/dialog/with-dialog';
import { COLLECTION_PARTIAL_COPY_FORM_NAME, copyCollectionPartial, CollectionPartialCopyFormData } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { DialogCollectionPartialCopy } from '~/views-components/dialog-copy/dialog-collection-partial-copy';

export const CopyCollectionPartialDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY_FORM_NAME),
    reduxForm<CollectionPartialCopyFormData>({
        form: COLLECTION_PARTIAL_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollectionPartial(data));
        }
    })
)(DialogCollectionPartialCopy);
