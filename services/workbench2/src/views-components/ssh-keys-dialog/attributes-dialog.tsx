// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { compose } from 'redux';
import { withStyles, Dialog, DialogTitle, DialogContent, DialogActions, Button, StyleRulesCallback, WithStyles, Grid } from '@material-ui/core';
import { WithDialogProps, withDialog } from "store/dialog/with-dialog";
import { SSH_KEY_ATTRIBUTES_DIALOG } from 'store/auth/auth-action-ssh';
import { ArvadosTheme } from 'common/custom-theme';
import { SshKeyResource } from "models/ssh-key";

type CssRules = 'root';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem',
        '& div:nth-child(odd)': {
            textAlign: 'right',
            color: theme.palette.grey["500"]
        }
    }
});

interface AttributesSshKeyDialogDataProps {
    sshKey: SshKeyResource;
}

export const AttributesSshKeyDialog = compose(
    withDialog(SSH_KEY_ATTRIBUTES_DIALOG),
    withStyles(styles))(
        ({ open, closeDialog, data, classes }: WithDialogProps<AttributesSshKeyDialogDataProps> & WithStyles<CssRules>) =>
            <Dialog open={open}
                onClose={closeDialog}
                fullWidth
                maxWidth='sm'>
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    {data.sshKey && <Grid container direction="row" spacing={16} className={classes.root}>
                        <Grid item xs={5}>Name</Grid>
                        <Grid item xs={7}>{data.sshKey.name}</Grid>
                        <Grid item xs={5}>uuid</Grid>
                        <Grid item xs={7}>{data.sshKey.uuid}</Grid>
                        <Grid item xs={5}>Owner uuid</Grid>
                        <Grid item xs={7}>{data.sshKey.ownerUuid}</Grid>
                        <Grid item xs={5}>Authorized user uuid</Grid>
                        <Grid item xs={7}>{data.sshKey.authorizedUserUuid}</Grid>
                        <Grid item xs={5}>Created at</Grid>
                        <Grid item xs={7}>{data.sshKey.createdAt}</Grid>
                        <Grid item xs={5}>Modified at</Grid>
                        <Grid item xs={7}>{data.sshKey.modifiedAt}</Grid>
                        <Grid item xs={5}>Expires at</Grid>
                        <Grid item xs={7}>{data.sshKey.expiresAt}</Grid>
                        <Grid item xs={5}>Modified by user uuid</Grid>
                        <Grid item xs={7}>{data.sshKey.modifiedByUserUuid}</Grid>
                        <Grid item xs={5}>Modified by client uuid</Grid>
                        <Grid item xs={7}>{data.sshKey.modifiedByClientUuid}</Grid>
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
