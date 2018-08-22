// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { withDialog } from "~/store/dialog/with-dialog";
import { reduxForm } from 'redux-form';
import { PROJECT_COPY_DIALOG } from '~/store/project-copy-project-dialog/project-copy-project-dialog';
import { ProjectCopyFormDialog } from "~/views-components/project-copy-dialog/project-copy-dialog";
import { copyProject } from '../../store/project-copy-project-dialog/project-copy-project-dialog';

export const ProjectCopyDialog = compose(
    withDialog(PROJECT_COPY_DIALOG),
    reduxForm({
        form: PROJECT_COPY_DIALOG,
        onSubmit: (data, dispatch) => dispatch(copyProject(data))
    })
)(ProjectCopyFormDialog);