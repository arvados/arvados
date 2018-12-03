// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Field, InjectedFormProps } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Button, Typography, Grid, Table, TableHead, TableRow, TableCell, TableBody, Tooltip, IconButton } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { User } from "~/models/user";

type CssRules = 'root' | 'gridItem' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    gridItem: {
        height: 45,
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

export const MY_ACCOUNT_FORM = 'myAccountForm';

type MyAccountPanelRootProps = InjectedFormProps<MyAccountPanelRootActionProps> & MyAccountPanelRootDataProps & WithStyles<CssRules>;

export const MyAccountPanelRoot = withStyles(styles)(
    ({ classes, user }: MyAccountPanelRootProps) => {
        console.log(user);
        return <Card className={classes.root}>
            <CardContent>
                <Typography variant="title" className={classes.title}>User profile</Typography>
                <Grid container direction="row" spacing={24}>
                    <Grid item xs={6}>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="E-mail"
                                name="email"
                                component={TextField}
                                value={user!.email}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="First name"
                                name="firstName"
                                component={TextField}
                                value={user!.firstName}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="Identity URL"
                                name="identityUrl"
                                component={TextField}
                                value={user!.identityUrl}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="Organization"
                                name="organization"
                                value={user!.prefs.profile!.organization}
                                component={TextField}
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="Website"
                                name="website"
                                value={user!.prefs.profile!.website_url}
                                component={TextField}
                            />
                        </Grid>
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="Role"
                                name="role"
                                value={user!.prefs.profile!.role}
                                component={TextField}
                            />
                        </Grid>
                    </Grid>
                    <Grid item xs={6}>
                        <Grid item className={classes.gridItem} />
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="Last name"
                                name="lastName"
                                component={TextField}
                                value={user!.lastName}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} />
                        <Grid item className={classes.gridItem}>
                            <Field
                                label="E-mail at Organization"
                                name="organizationEmail"
                                value={user!.prefs.profile!.organization_email}
                                component={TextField}
                            />
                        </Grid>
                    </Grid>
                </Grid>
            </CardContent>
        </Card>;}
);