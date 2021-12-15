// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProjectUpdateFormDialogData } from 'store/projects/project-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectNameField, ProjectDescriptionField, UsersField } from 'views-components/form-fields/project-form-fields';
import { GroupClass } from 'models/group';

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass, showUsersField?: boolean}> & InjectedFormProps<ProjectUpdateFormDialogData>;

export const DialogProjectUpdate = (props: DialogProjectProps) => {
    let title = 'Edit Project';
    let fields = ProjectEditFields;
    const sourcePanel = props.data.sourcePanel || '';
    const showUsersField = !!props.data.showUsersField;

    if (sourcePanel === GroupClass.ROLE) {
        title = showUsersField ? 'Create Group' : 'Edit Group';
        fields = showUsersField ? GroupAddFields : ProjectEditFields;
    }

    return <FormDialog
        dialogTitle={title}
        formFields={fields}
        submitLabel='Save'
        {...props}
    />;
};

// Also used as "Group Edit Fields"
const ProjectEditFields = () => <span>
    <ProjectNameField />
    <ProjectDescriptionField />
</span>;

const GroupAddFields = () => <span>
    <ProjectNameField />
    <UsersField />
    <ProjectDescriptionField />
</span>;
