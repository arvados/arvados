// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogProjectUpdate } from 'views-components/dialog-update/dialog-project-update';
import { PROJECT_UPDATE_FORM_NAME, ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { updateProject } from 'store/workbench/workbench-actions';

export const UpdateProjectDialog = compose(
    withDialog(PROJECT_UPDATE_FORM_NAME),
    reduxForm<ProjectUpdateFormDialogData>({
        form: PROJECT_UPDATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(updateProject(data));
        }
    })
)(DialogProjectUpdate);