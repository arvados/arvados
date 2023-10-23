// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { COLLECTION_COPY_FORM_NAME } from 'store/collections/collection-copy-actions';
import { DialogCopy } from "views-components/dialog-copy/dialog-copy";
import { copyCollection } from 'store/workbench/workbench-actions';
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { pickerId } from 'store/tree-picker/picker-id';

export const CopyCollectionDialog = compose(
    withDialog(COLLECTION_COPY_FORM_NAME),
    reduxForm<CopyFormDialogData>({
        form: COLLECTION_COPY_FORM_NAME,
        touchOnChange: true,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollection(data));
        }
    }),
    pickerId(COLLECTION_COPY_FORM_NAME),
)(DialogCopy);