// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectNameField, ProjectDescriptionField } from 'views-components/form-fields/project-form-fields';

type DialogProjectProps = WithDialogProps<{}> & InjectedFormProps<ProjectUpdateFormDialogData>;

export const DialogProjectUpdate = (props: DialogProjectProps) =>
    <FormDialog
        dialogTitle='Edit Project'
        formFields={ProjectEditFields}
        submitLabel='Save'
        {...props}
    />;

const ProjectEditFields = () => <span>
    <ProjectNameField />
    <ProjectDescriptionField />
</span>;
