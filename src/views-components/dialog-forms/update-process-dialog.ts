// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogProcessUpdate } from 'views-components/dialog-update/dialog-process-update';
import { PROCESS_UPDATE_FORM_NAME, ProcessUpdateFormDialogData } from 'store/processes/process-update-actions';
import { updateProcess } from "store/workbench/workbench-actions";

export const UpdateProcessDialog = compose(
    withDialog(PROCESS_UPDATE_FORM_NAME),
    reduxForm<ProcessUpdateFormDialogData>({
        form: PROCESS_UPDATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(updateProcess(data));
        }
    })
)(DialogProcessUpdate);