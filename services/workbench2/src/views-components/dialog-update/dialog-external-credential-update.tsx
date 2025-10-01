// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps } from 'redux-form';
import { Grid } from "@mui/material";
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { UpdateExternalCredentialFormDialogData } from 'store/external-credentials/external-credential-dialog-data';
import { ExternalCredentialNameField,
    ExternalCredentialDescriptionField,
    ExternalCredentialClassUpdateField,
    ExternalCredentialExternalIdField,
    ExternalCredentialSecretUpdateField,
    ExternalCredentialExpiresAtField,
    ExternalCredentialScopesField } from 'views-components/form-fields/external-credential-form-fields';

type CssRules = 'description';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    description: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(2),
    },
});

type DialogExternalCredentialProps = WithDialogProps<{}> & InjectedFormProps<UpdateExternalCredentialFormDialogData>;

export const DialogExternalCredentialUpdate = (props: DialogExternalCredentialProps) => {
    let title = 'Edit External Credential';

    return <FormDialog
        dialogTitle={title}
        formFields={ExternalCredentialEditFields as any}
        submitLabel='Save'
        {...props}
    />;
};

const ExternalCredentialEditFields = withStyles(styles)(
    ({ classes }: WithStyles<CssRules>) => <span>
        <ExternalCredentialNameField />
        <div className={classes.description}>
            <ExternalCredentialDescriptionField />
        </div>
        <Grid container direction={'row'} xs={12} spacing={2}>
            <Grid item xs={6}><ExternalCredentialClassUpdateField /></Grid>
            <Grid item xs={6}><ExternalCredentialExternalIdField /></Grid>
            <Grid item xs={6}><ExternalCredentialSecretUpdateField /></Grid>
            <Grid item xs={6}><ExternalCredentialExpiresAtField /></Grid>
            <Grid item xs={12}><ExternalCredentialScopesField /></Grid>
        </Grid>
    </span>);
