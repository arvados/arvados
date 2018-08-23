// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { COLLECTION_COPY_DIALOG, CopyFormDialogData } from '~/store/collection-copy-dialog/collection-copy-dialog';
import { CopyFormDialog } from "~/views-components/copy-dialog/copy-dialog";
import { copyCollection } from '~/store/collection-copy-dialog/collection-copy-dialog';

export const CollectionCopyDialog = compose(
    withDialog(COLLECTION_COPY_DIALOG),
    reduxForm<CopyFormDialogData>({
        form: COLLECTION_COPY_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollection(data));
        }
    })
)(CopyFormDialog);