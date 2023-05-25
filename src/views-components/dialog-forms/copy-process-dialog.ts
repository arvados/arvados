// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from 'redux';
import { withDialog } from 'store/dialog/with-dialog';
import { reduxForm } from 'redux-form';
import { PROCESS_COPY_FORM_NAME, MULTI_PROCESS_COPY_FORM_NAME } from 'store/processes/process-copy-actions';
import { DialogProcessRerun, DialogManyProcessesRerun } from 'views-components/dialog-copy/dialog-process-rerun';
import { copyProcess } from 'store/workbench/workbench-actions';
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { pickerId } from 'store/tree-picker/picker-id';

export const CopyProcessDialog = compose(
    withDialog(PROCESS_COPY_FORM_NAME),
    reduxForm<CopyFormDialogData>({
        form: PROCESS_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyProcess(data));
        },
    }),
    pickerId(PROCESS_COPY_FORM_NAME)
)(DialogProcessRerun);

export const CopyManyProcessesDialog = compose(
    withDialog(MULTI_PROCESS_COPY_FORM_NAME),
    reduxForm<CopyFormDialogData>({
        form: MULTI_PROCESS_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            console.log('COPYMANY', data);
            dispatch(copyProcess(data));
        },
    }),
    pickerId(MULTI_PROCESS_COPY_FORM_NAME)
)(DialogManyProcessesRerun);
