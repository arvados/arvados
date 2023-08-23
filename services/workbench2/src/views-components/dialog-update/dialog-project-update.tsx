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
import { FormGroup, FormLabel, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { UpdateProjectPropertiesForm } from 'views-components/project-properties/update-project-properties-form';
import { resourcePropertiesList } from 'views-components/resource-properties/resource-properties-list';

type CssRules = 'propertiesForm' | 'description';

const styles: StyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing.unit * 2,
        marginBottom: theme.spacing.unit * 2,
    },
    description: {
        marginTop: theme.spacing.unit * 2,
        marginBottom: theme.spacing.unit * 2,
    },
});

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass}> & InjectedFormProps<ProjectUpdateFormDialogData>;

export const DialogProjectUpdate = (props: DialogProjectProps) => {
    let title = 'Edit Project';
    const sourcePanel = props.data.sourcePanel || '';

    if (sourcePanel === GroupClass.ROLE) {
        title = 'Edit Group';
    }

    return <FormDialog
        dialogTitle={title}
        formFields={ProjectEditFields as any}
        submitLabel='Save'
        {...props}
    />;
};

const UpdateProjectPropertiesList = resourcePropertiesList(PROJECT_UPDATE_FORM_NAME);

// Also used as "Group Edit Fields"
const ProjectEditFields = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) => <span>
        <ProjectNameField />
        <div className={classes.description}>
            <ProjectDescriptionField />
        </div>
        <div className={classes.propertiesForm}>
            <FormLabel>Properties</FormLabel>
            <FormGroup>
                <UpdateProjectPropertiesForm />
                <UpdateProjectPropertiesList />
            </FormGroup>
        </div>
    </span>);
