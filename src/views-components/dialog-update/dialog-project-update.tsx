// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProjectUpdateFormDialogData, PROJECT_UPDATE_FORM_NAME } from 'store/projects/project-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectNameField, ProjectDescriptionField } from 'views-components/form-fields/project-form-fields';
import { GroupClass } from 'models/group';
import { FormGroup, FormLabel } from '@material-ui/core';
import { UpdateProjectPropertiesForm } from 'views-components/project-properties/update-project-properties-form';
import { resourcePropertiesList } from 'views-components/resource-properties/resource-properties-list';

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass}> & InjectedFormProps<ProjectUpdateFormDialogData>;

export const DialogProjectUpdate = (props: DialogProjectProps) => {
    let title = 'Edit Project';
    const sourcePanel = props.data.sourcePanel || '';

    if (sourcePanel === GroupClass.ROLE) {
        title = 'Edit Group';
    }

    return <FormDialog
        dialogTitle={title}
        formFields={ProjectEditFields}
        submitLabel='Save'
        {...props}
    />;
};

const UpdateProjectPropertiesList = resourcePropertiesList(PROJECT_UPDATE_FORM_NAME);

// Also used as "Group Edit Fields"
const ProjectEditFields = () => <span>
    <ProjectNameField />
    <ProjectDescriptionField />
    <FormLabel>Properties</FormLabel>
    <FormGroup>
        <UpdateProjectPropertiesForm />
        <UpdateProjectPropertiesList />
    </FormGroup>
</span>;
