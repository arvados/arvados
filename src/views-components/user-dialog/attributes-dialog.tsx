// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, Grid } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from "redux";
import { USER_ATTRIBUTES_DIALOG } from "~/store/users/users-actions";
import { UserResource } from "~/models/user";

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

interface UserAttributesDataProps {
    data: UserResource;
}

type UserAttributesProps = UserAttributesDataProps & WithStyles<CssRules>;

export const UserAttributesDialog = compose(
    withDialog(USER_ATTRIBUTES_DIALOG),
    styles)(
        (props: WithDialogProps<UserAttributesProps> & UserAttributesProps) =>
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

const attributes = (user: UserResource, classes: any) => {
    const { uuid, ownerUuid, createdAt, modifiedAt, modifiedByClientUuid, modifiedByUserUuid,
        firstName, lastName, username, email, isActive, isAdmin } = user;
    return (
        <span>
            <Grid container direction="row">
                <Grid item xs={5} className={classes.rightContainer}>
                    {uuid && <Grid item>Uuid</Grid>}
                    {firstName && <Grid item>First name</Grid>}
                    {lastName && <Grid item>Last name</Grid>}
                    {email && <Grid item>Email</Grid>}
                    {username && <Grid item>Username</Grid>}
                    {isActive && <Grid item>Is active</Grid>}
                    {isAdmin && <Grid item>Is admin</Grid>}
                    {createdAt && <Grid item>Created at</Grid>}
                    {modifiedAt && <Grid item>Modified at</Grid>}
                    {ownerUuid && <Grid item>Owner uuid</Grid>}
                    {modifiedByUserUuid && <Grid item>Modified by user uuid</Grid>}
                    {modifiedByClientUuid && <Grid item>Modified by client uuid</Grid>}
                </Grid>
                <Grid item xs={7} className={classes.leftContainer}>
                    <Grid item>{uuid}</Grid>
                    <Grid item>{firstName}</Grid>
                    <Grid item>{lastName}</Grid>
                    <Grid item>{email}</Grid>
                    <Grid item>{username}</Grid>
                    <Grid item>{isActive}</Grid>
                    <Grid item>{isAdmin}</Grid>
                    <Grid item>{createdAt}</Grid>
                    <Grid item>{modifiedAt}</Grid>
                    <Grid item>{ownerUuid}</Grid>
                    <Grid item>{modifiedByUserUuid}</Grid>
                    <Grid item>{modifiedByClientUuid}</Grid>
                </Grid>
            </Grid>
        </span>
    );
};
