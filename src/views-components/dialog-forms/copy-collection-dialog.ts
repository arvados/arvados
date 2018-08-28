// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { COLLECTION_COPY_FORM_NAME, CollectionCopyFormDialogData } from '~/store/collections/collection-copy-actions';
import { DialogCollectionCopy } from "~/views-components/dialog-copy/dialog-collection-copy";
import { copyCollection } from '~/store/workbench/workbench-actions';

export const CopyCollectionDialog = compose(
    withDialog(COLLECTION_COPY_FORM_NAME),
    reduxForm<CollectionCopyFormDialogData>({
        form: COLLECTION_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollection(data));
        }
    })
)(DialogCollectionCopy);