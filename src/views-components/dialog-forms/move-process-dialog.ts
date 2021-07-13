// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from 'redux';
import { withDialog } from "store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { PROCESS_MOVE_FORM_NAME } from 'store/processes/process-move-actions';
import { MoveToFormDialogData } from 'store/move-to-dialog/move-to-dialog';
import { DialogMoveTo } from 'views-components/dialog-move/dialog-move-to';
import { moveProcess } from 'store/workbench/workbench-actions';
import { pickerId } from 'store/tree-picker/picker-id';

export const MoveProcessDialog = compose(
    withDialog(PROCESS_MOVE_FORM_NAME),
    reduxForm<MoveToFormDialogData>({
        form: PROCESS_MOVE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(moveProcess(data));
        }
    }),
    pickerId(PROCESS_MOVE_FORM_NAME),
)(DialogMoveTo);