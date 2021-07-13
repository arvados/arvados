// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, Grid } from "@material-ui/core";
import { WithDialogProps } from "store/dialog/with-dialog";
import { withDialog } from 'store/dialog/with-dialog';
import { WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { compose } from "redux";
import { PermissionResource } from "models/permission";
import { MEMBER_ATTRIBUTES_DIALOG } from 'store/group-details-panel/group-details-panel-actions';

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

interface GroupAttributesDataProps {
    data: PermissionResource;
}

type GroupAttributesProps = GroupAttributesDataProps & WithStyles<CssRules>;

export const GroupMemberAttributesDialog = compose(
    withDialog(MEMBER_ATTRIBUTES_DIALOG),
    styles)(
        (props: WithDialogProps<GroupAttributesProps> & GroupAttributesProps) =>
            <Dialog open={props.open}
                onClose={props.closeDialog}
                fullWidth
                maxWidth="sm">
                <DialogTitle>Attributes</DialogTitle>
                <DialogContent>
                    <Typography variant='body1' className={props.classes.spacing}>
                        {props.data && attributes(props.data, props.classes)}
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

const attributes = (memberGroup: PermissionResource, classes: any) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid, name, etag, href, linkClass } = memberGroup;
    return (
        <span>
            <Grid container direction="row">
                <Grid item xs={5} className={classes.rightContainer}>
                    {name && <Grid item>Name</Grid>}
                    {ownerUuid && <Grid item>Owner uuid</Grid>}
                    {createdAt && <Grid item>Created at</Grid>}
                    {modifiedAt && <Grid item>Modified at</Grid>}
                    {modifiedByUserUuid && <Grid item>Modified by user uuid</Grid>}
                    {modifiedByClientUuid && <Grid item>Modified by client uuid</Grid>}
                    {uuid && <Grid item>uuid</Grid>}
                    {linkClass && <Grid item>Link Class</Grid>}
                    {etag && <Grid item>Etag</Grid>}
                    {href && <Grid item>Href</Grid>}
                </Grid>
                <Grid item xs={7} className={classes.leftContainer}>
                    <Grid item>{name}</Grid>
                    <Grid item>{ownerUuid}</Grid>
                    <Grid item>{createdAt}</Grid>
                    <Grid item>{modifiedAt}</Grid>
                    <Grid item>{modifiedByUserUuid}</Grid>
                    <Grid item>{modifiedByClientUuid}</Grid>
                    <Grid item>{uuid}</Grid>
                    <Grid item>{linkClass}</Grid>
                    <Grid item>{etag}</Grid>
                    <Grid item>{href}</Grid>
                </Grid>
            </Grid>
        </span>
    );
};
