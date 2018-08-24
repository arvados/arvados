// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { COLLECTION_COPY_DIALOG, CollectionCopyFormDialogData, copyCollection } from '~/store/collections/copy/collection-copy-actions';
import { DialogCopy } from "~/views-components/dialog-copy/dialog-copy";

export const CopyCollectionDialog = compose(
    withDialog(COLLECTION_COPY_DIALOG),
    reduxForm<CollectionCopyFormDialogData>({
        form: COLLECTION_COPY_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollection(data));
        }
    })
)(DialogCopy);