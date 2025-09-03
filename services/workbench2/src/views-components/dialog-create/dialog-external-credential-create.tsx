// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Grid } from "@mui/material";
import { WithDialogProps } from 'store/dialog/with-dialog';
import { ProjectCreateFormDialogData } from 'store/projects/project-create-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ExternalCredentialNameField, ExternalCredentialDescriptionField, ExternalCredentialClassField, ExternalCredentialExternalIdField, ExternalCredentialExpiresAtField, ExternalCredentialScopesField } from 'views-components/form-fields/external-credential-form-fields';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { GroupClass } from 'models/group';

type CssRules = 'propertiesForm' | 'description';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    propertiesForm: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
    description: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass}> & InjectedFormProps<ProjectCreateFormDialogData>;

export const DialogExternalCredentialCreate = (props: DialogProjectProps) => {
    const title = 'New External Credential';
    const fields = NewExternalCredentialFields;

    return <FormDialog
        dialogTitle={title}
        formFields={fields as any}
        submitLabel='Create'
        {...props}
    />;
};

const NewExternalCredentialFields = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) => <span>
        <ExternalCredentialNameField />
        <div className={classes.description}>
            <ExternalCredentialDescriptionField />
        </div>
        <Grid container direction={'row'} xs={12} spacing={2}>
            <Grid item xs={6}><ExternalCredentialClassField /></Grid>
            <Grid item xs={6}><ExternalCredentialExternalIdField /></Grid>
            <Grid item xs={6}><ExternalCredentialExpiresAtField /></Grid>
            <Grid item xs={6}><ExternalCredentialScopesField /></Grid>
        </Grid>
    </span>);


