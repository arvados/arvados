// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { PROJECT_MOVE_FORM_NAME } from '~/store/projects/project-move-actions';
import { MoveToFormDialogData } from '~/store/move-to-dialog/move-to-dialog';
import { DialogMoveTo } from '~/views-components/dialog-move/dialog-move-to';
import { moveProject } from '~/store/workbench/workbench-actions';
import { pickerId } from '~/store/tree-picker/picker-id';

export const MoveProjectDialog = compose(
    withDialog(PROJECT_MOVE_FORM_NAME),
    reduxForm<MoveToFormDialogData>({
        form: PROJECT_MOVE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(moveProject(data));
        }
    }),
    pickerId(PROJECT_MOVE_FORM_NAME),
)(DialogMoveTo);

