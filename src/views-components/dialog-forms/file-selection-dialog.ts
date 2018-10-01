// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { FILE_SELECTION } from '~/store/file-selection/file-selection-actions';
import { DialogFileSelection } from '~/views-components/dialog-file-selection/dialog-file-selection';
import { dialogActions } from '~/store/dialog/dialog-actions';

export const FileSelectionDialog = compose(
    withDialog(FILE_SELECTION),
    reduxForm({
        form: FILE_SELECTION,
        onSubmit: (data, dispatch) => {
            dispatch(dialogActions.CLOSE_DIALOG({ id: FILE_SELECTION }));
            return data;
        }
    })
)(DialogFileSelection);