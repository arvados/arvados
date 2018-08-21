// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { MOVE_PROJECT_DIALOG } from '~/store/move-project-dialog/move-project-dialog';
import { moveProject } from '~/store/move-project-dialog/move-project-dialog';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { MoveToFormDialog } from '../move-to-dialog/move-to-dialog';

export const MoveProjectDialog = compose(
    withDialog(MOVE_PROJECT_DIALOG),
    reduxForm<MoveToFormDialogData>({
        form: MOVE_PROJECT_DIALOG,
        onSubmit: (data, dispatch) => {
            dispatch(moveProject(data));
        }
    })
)(MoveToFormDialog);

