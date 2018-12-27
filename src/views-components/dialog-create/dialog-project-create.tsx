// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { ProjectCreateFormDialogData } from '~/store/projects/project-create-actions';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { ProjectNameField, ProjectDescriptionField } from '~/views-components/form-fields/project-form-fields';
import { CreateProjectPropertiesForm } from '~/views-components/project-properties/create-project-properties-form';
import { CreateProjectPropertiesList } from '~/views-components/project-properties/create-project-properties-list';

type DialogProjectProps = WithDialogProps<{}> & InjectedFormProps<ProjectCreateFormDialogData>;

export const DialogProjectCreate = (props: DialogProjectProps) =>
    <FormDialog
        dialogTitle='New project'
        formFields={ProjectAddFields}
        submitLabel='Create a Project'
        {...props}
    />;

const ProjectAddFields = () => <span>
    <ProjectNameField />
    <ProjectDescriptionField />
    <CreateProjectPropertiesForm />
    <CreateProjectPropertiesList />
</span>;
