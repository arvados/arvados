// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { DialogMoveTo } from '~/views-components/dialog-move/dialog-move-to';
import { COLLECTION_MOVE_FORM_NAME } from '~/store/collections/collection-move-actions';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { moveCollection } from '~/store/workbench/workbench-actions';
import { pickerId } from '~/store/tree-picker/picker-id';

export const MoveCollectionDialog = compose(
    withDialog(COLLECTION_MOVE_FORM_NAME),
    reduxForm<MoveToFormDialogData>({
        form: COLLECTION_MOVE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(moveCollection(data));
        }
    }),
    pickerId(COLLECTION_MOVE_FORM_NAME),
)(DialogMoveTo);
