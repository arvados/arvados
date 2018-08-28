// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { DialogCollectionUpdate } from '~/views-components/dialog-update/dialog-collection-update';
import { COLLECTION_UPDATE_FORM_NAME, CollectionUpdateFormDialogData } from '~/store/collections/collection-update-actions';
import { updateCollection } from "~/store/workbench/workbench-actions";

export const UpdateCollectionDialog = compose(
    withDialog(COLLECTION_UPDATE_FORM_NAME),
    reduxForm<CollectionUpdateFormDialogData>({
        form: COLLECTION_UPDATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(updateCollection(data));
        }
    })
)(DialogCollectionUpdate);