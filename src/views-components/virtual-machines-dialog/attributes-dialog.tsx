// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, Grid } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { VIRTUAL_MACHINE_ATTRIBUTES_DIALOG } from "~/store/virtual-machines/virtual-machines-actions";
import { WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from "redux";
import { VirtualMachinesResource } from "~/models/virtual-machines";

type CssRules = 'rightContainer' | 'leftContainer' | 'spacing';

const styles = withStyles<CssRules>((theme: ArvadosTheme) => ({
    rightContainer: {
        textAlign: 'right',
        paddingRight: theme.spacing.unit * 2,
        color: theme.palette.grey["500"]
    },
    leftContainer: {
        textAlign: 'left',
        paddingLeft: theme.spacing.unit * 2
    },
    spacing: {
        paddingTop: theme.spacing.unit * 2
    },
}));

interface VirtualMachineAttributesDataProps {
    virtualMachineData: VirtualMachinesResource;
}

type VirtualMachineAttributesProps = VirtualMachineAttributesDataProps & WithStyles<CssRules>;

export const VirtualMachineAttributesDialog = compose(
    withDialog(VIRTUAL_MACHINE_ATTRIBUTES_DIALOG),
    styles)(
        (props: WithDialogProps<VirtualMachineAttributesProps> & VirtualMachineAttributesProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth="sm">
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    <Typography variant='body1' className={props.classes.spacing}>
                        {props.data.virtualMachineData && attributes(props.data.virtualMachineData, props.classes)}
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={props.closeDialog}>
                        Close
                </Button>
                </DialogActions>
            </Dialog>
    );

const attributes = (virtualMachine: VirtualMachinesResource, classes: any) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, hostname } = virtualMachine;
    return (
        <span>
            <Grid container direction="row">
                <Grid item xs={5} className={classes.rightContainer}>
                    <Grid item>Hostname</Grid>
                    <Grid item>Owner uuid</Grid>
                    <Grid item>Created at</Grid>
                    <Grid item>Modified at</Grid>
                    <Grid item>Modified by user uuid</Grid>
                    <Grid item>Modified by client uuid</Grid>
                    <Grid item>uuid</Grid>
                </Grid>
                <Grid item xs={7} className={classes.leftContainer}>
                    <Grid item>{hostname}</Grid>
                    <Grid item>{ownerUuid}</Grid>
                    <Grid item>{createdAt}</Grid>
                    <Grid item>{modifiedAt}</Grid>
                    <Grid item>{modifiedByUserUuid}</Grid>
                    <Grid item>{modifiedByClientUuid}</Grid>
                    <Grid item>{uuid}</Grid>
                </Grid>
            </Grid>
        </span>
    );
};
