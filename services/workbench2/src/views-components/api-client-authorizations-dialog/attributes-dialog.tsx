// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { compose } from 'redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Grid } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { WithDialogProps, withDialog } from "store/dialog/with-dialog";
import { API_CLIENT_AUTHORIZATION_ATTRIBUTES_DIALOG } from 'store/api-client-authorizations/api-client-authorizations-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { ApiClientAuthorization } from 'models/api-client-authorization';
import { formatDateTime } from 'common/formatters';

type CssRules = 'root';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem',
        '& div:nth-child(odd)': {
            textAlign: 'right',
            color: theme.palette.grey["500"]
        }
    }
});

interface AttributesKeepServiceDialogDataProps {
    apiClientAuthorization: ApiClientAuthorization;
}

export const AttributesApiClientAuthorizationDialog = compose(
    withDialog(API_CLIENT_AUTHORIZATION_ATTRIBUTES_DIALOG),
    withStyles(styles))(
        ({ open, closeDialog, data, classes }: WithDialogProps<AttributesKeepServiceDialogDataProps> & WithStyles<CssRules>) =>
            <Dialog open={open} onClose={closeDialog} fullWidth maxWidth='sm'>
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    {data.apiClientAuthorization && <Grid container direction="row" spacing={2} className={classes.root}>
                        <Grid item xs={5}>UUID</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.uuid}</Grid>
                        <Grid item xs={5}>Owner uuid</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.ownerUuid}</Grid>
                        <Grid item xs={5}>API Token</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.apiToken}</Grid>
                        <Grid item xs={5}>Created by IP address</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.createdByIpAddress || '(none)'}</Grid>
                        <Grid item xs={5}>Expires at</Grid>
                        <Grid item xs={7}>{formatDateTime(data.apiClientAuthorization.expiresAt) || '(none)'}</Grid>
                        <Grid item xs={5}>Last used at</Grid>
                        <Grid item xs={7}>{formatDateTime(data.apiClientAuthorization.lastUsedAt) || '(none)'}</Grid>
                        <Grid item xs={5}>Last used by IP address</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.lastUsedByIpAddress || '(none)'}</Grid>
                        <Grid item xs={5}>Scopes</Grid>
                        <Grid item xs={7}>{JSON.stringify(data.apiClientAuthorization.scopes || '(none)')}</Grid>
                        <Grid item xs={5}>User ID</Grid>
                        <Grid item xs={7}>{data.apiClientAuthorization.userId || '(none)'}</Grid>
                        <Grid item xs={5}>Created at</Grid>
                        <Grid item xs={7}>{formatDateTime(data.apiClientAuthorization.createdAt) || '(none)'}</Grid>
                        <Grid item xs={5}>Updated at</Grid>
                        <Grid item xs={7}>{formatDateTime(data.apiClientAuthorization.updatedAt) || '(none)'}</Grid>
                    </Grid>}
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
    );
