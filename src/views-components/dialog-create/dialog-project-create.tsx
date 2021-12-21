// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProjectCreateFormDialogData, PROJECT_CREATE_FORM_NAME } from 'store/projects/project-create-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectNameField, ProjectDescriptionField, UsersField } from 'views-components/form-fields/project-form-fields';
import { CreateProjectPropertiesForm } from 'views-components/project-properties/create-project-properties-form';
import { ResourceParentField } from '../form-fields/resource-form-fields';
import { FormGroup, FormLabel } from '@material-ui/core';
import { resourcePropertiesList } from 'views-components/resource-properties/resource-properties-list';
import { GroupClass } from 'models/group';

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass}> & InjectedFormProps<ProjectCreateFormDialogData>;

export const DialogProjectCreate = (props: DialogProjectProps) => {
    let title = 'New Project';
    let fields = ProjectAddFields;
    const sourcePanel = props.data.sourcePanel || '';

    if (sourcePanel === GroupClass.ROLE) {
        title = 'New Group';
        fields = GroupAddFields;
    }

    return <FormDialog
        dialogTitle={title}
        formFields={fields}
        submitLabel='Create'
        {...props}
    />;
};

const CreateProjectPropertiesList = resourcePropertiesList(PROJECT_CREATE_FORM_NAME);

const ProjectAddFields = () => <span>
    <ResourceParentField />
    <ProjectNameField />
    <ProjectDescriptionField />
    <FormLabel>Properties</FormLabel>
    <FormGroup>
        <CreateProjectPropertiesForm />
        <CreateProjectPropertiesList />
    </FormGroup>
</span>;

const GroupAddFields = () => <span>
    <ProjectNameField />
    <UsersField />
    <ProjectDescriptionField />
    <FormLabel>Properties</FormLabel>
    <FormGroup>
        <CreateProjectPropertiesForm />
        <CreateProjectPropertiesList />
    </FormGroup>
</span>;
