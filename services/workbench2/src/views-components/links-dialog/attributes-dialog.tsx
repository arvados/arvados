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
import { LINK_ATTRIBUTES_DIALOG } from 'store/link-panel/link-panel-actions';
import { ArvadosTheme } from 'common/custom-theme';
import { LinkResource } from 'models/link';

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

interface AttributesLinkDialogDataProps {
    link: LinkResource;
}

export const AttributesLinkDialog = compose(
    withDialog(LINK_ATTRIBUTES_DIALOG),
    withStyles(styles))(
    ({ open, closeDialog, data, classes }: WithDialogProps<AttributesLinkDialogDataProps> & WithStyles<CssRules>) =>
            <Dialog open={open}
                onClose={closeDialog}
                fullWidth
                maxWidth='sm'>
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    {data.link && <Grid container direction="row" spacing={2} className={classes.root}>
                        <Grid item xs={5}>Uuid</Grid>
                        <Grid item xs={7}>{data.link.uuid}</Grid>
                        <Grid item xs={5}>Name</Grid>
                        <Grid item xs={7}>{data.link.name}</Grid>
                        <Grid item xs={5}>Head uuid</Grid>
                        <Grid item xs={7}>{data.link.headUuid}</Grid>
                        <Grid item xs={5}>Head kind</Grid>
                        <Grid item xs={7}>{data.link.headKind}</Grid>
                        <Grid item xs={5}>Tail uuid</Grid>
                        <Grid item xs={7}>{data.link.tailUuid}</Grid>
                        <Grid item xs={5}>Link class</Grid>
                        <Grid item xs={7}>{data.link.linkClass}</Grid>
                        <Grid item xs={5}>Owner uuid</Grid>
                        <Grid item xs={7}>{data.link.ownerUuid}</Grid>
                        <Grid item xs={5}>Created at</Grid>
                        <Grid item xs={7}>{data.link.createdAt}</Grid>
                        <Grid item xs={5}>Modified at</Grid>
                        <Grid item xs={7}>{data.link.modifiedAt}</Grid>
                        <Grid item xs={5}>Modified by user uuid</Grid>
                        <Grid item xs={7}>{data.link.modifiedByUserUuid}</Grid>
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
