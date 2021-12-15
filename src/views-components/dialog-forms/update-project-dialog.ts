// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { DialogProjectUpdate } from 'views-components/dialog-update/dialog-project-update';
import { PROJECT_UPDATE_FORM_NAME, ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { updateProject, updateGroup } from 'store/workbench/workbench-actions';
import { GroupClass } from "models/group";
import { createGroup } from "store/groups-panel/groups-panel-actions";

export const UpdateProjectDialog = compose(
    withDialog(PROJECT_UPDATE_FORM_NAME),
    reduxForm<ProjectUpdateFormDialogData>({
        form: PROJECT_UPDATE_FORM_NAME,
        onSubmit: (data, dispatch, props) => {
            switch (props.data.sourcePanel) {
                case GroupClass.PROJECT:
                    dispatch(updateProject(data));
                    break;
                case GroupClass.ROLE:
                    if (data.uuid) {
                        dispatch(updateGroup(data));
                    } else {
                        dispatch(createGroup(data));
                    }
                    break;
                default:
                    break;
            }
        }
    })
)(DialogProjectUpdate);
