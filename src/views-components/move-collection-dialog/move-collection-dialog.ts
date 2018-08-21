// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { MoveToFormDialog } from '../move-to-dialog/move-to-dialog';
import { MOVE_COLLECTION_DIALOG, moveCollection } from '~/store/move-collection-dialog/move-collection-dialog';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';

export const MoveCollectionDialog = compose(
    withDialog(MOVE_COLLECTION_DIALOG),
    reduxForm<MoveToFormDialogData>({
        form: MOVE_COLLECTION_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(moveCollection(data));
        }
    })
)(MoveToFormDialog);
