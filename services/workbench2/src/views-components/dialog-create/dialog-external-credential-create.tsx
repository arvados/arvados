// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Grid } from "@mui/material";
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ExternalCredentialNameField,
    ExternalCredentialDescriptionField,
    ExternalCredentialClassCreateField,
    ExternalCredentialExternalIdField,
    ExternalCredentialExpiresAtField,
    ExternalCredentialSecretCreateField,
    ExternalCredentialScopesField } from 'views-components/form-fields/external-credential-form-fields';
import { CreateExternalCredentialFormDialogData } from 'store/external-credentials/external-credential-dialog-data';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { GroupClass } from 'models/group';

type CssRules = 'description';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    description: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

type DialogProjectProps = WithDialogProps<{sourcePanel: GroupClass}> & InjectedFormProps<CreateExternalCredentialFormDialogData>;

export const DialogExternalCredentialCreate = (props: DialogProjectProps) => {
    const title = 'New External Credential';

    return <FormDialog
        dialogTitle={title}
        formFields={NewExternalCredentialFields as any}
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
            <Grid item xs={6}><ExternalCredentialClassCreateField /></Grid>
            <Grid item xs={6}><ExternalCredentialExternalIdField /></Grid>
            <Grid item xs={6}><ExternalCredentialSecretCreateField /></Grid>
            <Grid item xs={6}><ExternalCredentialExpiresAtField /></Grid>
            <Grid item xs={12}><ExternalCredentialScopesField /></Grid>
        </Grid>
    </span>);


