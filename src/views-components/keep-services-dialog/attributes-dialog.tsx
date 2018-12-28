// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { compose } from 'redux';
import {
    withStyles, Dialog, DialogTitle, DialogContent, DialogActions,
    Button, StyleRulesCallback, WithStyles, Grid
} from '@material-ui/core';
import { WithDialogProps, withDialog } from "~/store/dialog/with-dialog";
import { KEEP_SERVICE_ATTRIBUTES_DIALOG } from '~/store/keep-services/keep-services-actions';
import { ArvadosTheme } from '~/common/custom-theme';
import { KeepServiceResource } from '~/models/keep-services';

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

interface AttributesKeepServiceDialogDataProps {
    keepService: KeepServiceResource;
}

export const AttributesKeepServiceDialog = compose(
    withDialog(KEEP_SERVICE_ATTRIBUTES_DIALOG),
    withStyles(styles))(
        ({ open, closeDialog, data, classes }: WithDialogProps<AttributesKeepServiceDialogDataProps> & WithStyles<CssRules>) =>
            <Dialog open={open} onClose={closeDialog} fullWidth maxWidth='sm'>
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    {data.keepService && <Grid container direction="row" spacing={16} className={classes.root}>
                        <Grid item xs={5}>UUID</Grid>
                        <Grid item xs={7}>{data.keepService.uuid}</Grid>
                        <Grid item xs={5}>Read only</Grid>
                        <Grid item xs={7}>{JSON.stringify(data.keepService.readOnly)}</Grid>
                        <Grid item xs={5}>Service host</Grid>
                        <Grid item xs={7}>{data.keepService.serviceHost}</Grid>
                        <Grid item xs={5}>Service port</Grid>
                        <Grid item xs={7}>{data.keepService.servicePort}</Grid>
                        <Grid item xs={5}>Service SSL flag</Grid>
                        <Grid item xs={7}>{JSON.stringify(data.keepService.serviceSslFlag)}</Grid>
                        <Grid item xs={5}>Service type</Grid>
                        <Grid item xs={7}>{data.keepService.serviceType}</Grid>
                        <Grid item xs={5}>Owner uuid</Grid>
                        <Grid item xs={7}>{data.keepService.ownerUuid}</Grid>
                        <Grid item xs={5}>Created at</Grid>
                        <Grid item xs={7}>{data.keepService.createdAt}</Grid>
                        <Grid item xs={5}>Modified at</Grid>
                        <Grid item xs={7}>{data.keepService.modifiedAt}</Grid>
                        <Grid item xs={5}>Modified by user uuid</Grid>
                        <Grid item xs={7}>{data.keepService.modifiedByUserUuid}</Grid>
                        <Grid item xs={5}>Modified by client uuid</Grid>
                        <Grid item xs={7}>{data.keepService.modifiedByClientUuid}</Grid>
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