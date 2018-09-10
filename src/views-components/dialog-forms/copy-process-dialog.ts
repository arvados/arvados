// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { PROCESS_COPY_FORM_NAME, ProcessCopyFormDialogData } from '~/store/processes/process-copy-actions';
import { DialogCopy } from "~/views-components/dialog-copy/dialog-collection-copy";
import { copyCollection } from '~/store/workbench/workbench-actions';

export const CopyProcessDialog = compose(
    withDialog(PROCESS_COPY_FORM_NAME),
    reduxForm<ProcessCopyFormDialogData>({
        form: PROCESS_COPY_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(copyCollection(data));
        }
    })
)(DialogCopy);