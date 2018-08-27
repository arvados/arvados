// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { addProject, PROJECT_CREATE_FORM_NAME, ProjectCreateFormDialogData } from '~/store/projects/project-create-actions';
import { DialogProjectCreate } from '~/views-components/dialog-create/dialog-project-create';

export const CreateProjectDialog = compose(
    withDialog(PROJECT_CREATE_FORM_NAME),
    reduxForm<ProjectCreateFormDialogData>({
        form: PROJECT_CREATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(addProject(data));
        }
    })
)(DialogProjectCreate);