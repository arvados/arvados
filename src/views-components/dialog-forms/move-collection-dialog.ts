// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { MoveToFormDialog } from '~/views-components/dialog-move/move-to-dialog';
import { COLLECTION_MOVE_FORM_NAME, moveCollection } from '~/store/collections/collection-move-actions';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';

export const MoveCollectionDialog = compose(
    withDialog(COLLECTION_MOVE_FORM_NAME),
    reduxForm<MoveToFormDialogData>({
        form: COLLECTION_MOVE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(moveCollection(data));
        }
    })
)(MoveToFormDialog);
