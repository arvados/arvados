// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles, Card, CardContent, TextField, Button, Typography, Grid, Table, TableHead, TableRow, TableCell, TableBody, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { User } from "~/models/user";

type CssRules = 'root' | 'gridItem' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    gridItem: {
        minHeight: 45,
        marginBottom: 20
    },
    title: {
        marginBottom: theme.spacing.unit * 3,
        color: theme.palette.grey["600"]
    }
});

export interface MyAccountPanelRootActionProps {}

export interface MyAccountPanelRootDataProps {
    user?: User;
}

type MyAccountPanelRootProps = MyAccountPanelRootActionProps & MyAccountPanelRootDataProps & WithStyles<CssRules>;

export const MyAccountPanelRoot = withStyles(styles)(
    ({ classes, user }: MyAccountPanelRootProps) => {
        console.log(user);
        return <Card className={classes.root}>
            <CardContent>
                <Typography variant="title" className={classes.title}>User profile</Typography>
                <Grid container direction="row" spacing={24}>
                    <Grid item xs={6}>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="E-mail"
                                name="email"
                                fullWidth
                                value={user!.email}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="First name"
                                name="firstName"
                                fullWidth
                                value={user!.firstName}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="Identity URL"
                                name="identityUrl"
                                fullWidth
                                value={user!.identityUrl}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="Organization"
                                name="organization"
                                value={user!.prefs.profile!.organization}
                                fullWidth
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="Website"
                                name="website"
                                value={user!.prefs.profile!.website_url}
                                fullWidth
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="Role"
                                name="role"
                                value={user!.prefs.profile!.role}
                                fullWidth
                            />
                        </Grid>
                    </Grid>
                    <Grid item xs={6}>
                        <Grid item className={classes.gridItem} />
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="Last name"
                                name="lastName"
                                fullWidth
                                value={user!.lastName}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} />
                        <Grid item className={classes.gridItem}>
                            <TextField
                                label="E-mail at Organization"
                                name="organizationEmail"
                                value={user!.prefs.profile!.organization_email}
                                fullWidth
                            />
                        </Grid>
                    </Grid>
                </Grid>
            </CardContent>
        </Card>;}
);