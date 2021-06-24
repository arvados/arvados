// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { PROCESS_COPY_FORM_NAME } from 'store/processes/process-copy-actions';
import { DialogCopy } from "views-components/dialog-copy/dialog-copy";
import { copyProcess } from 'store/workbench/workbench-actions';
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { pickerId } from "store/tree-picker/picker-id";

export const CopyProcessDialog = compose(
    withDialog(PROCESS_COPY_FORM_NAME),
    reduxForm<CopyFormDialogData>({
        form: PROCESS_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyProcess(data));
        }
    }),
    pickerId(PROCESS_COPY_FORM_NAME),
)(DialogCopy);