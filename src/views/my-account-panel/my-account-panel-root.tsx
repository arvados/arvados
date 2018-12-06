// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Field, InjectedFormProps } from "redux-form";
import { TextField } from "~/components/text-field/text-field";
import { NativeSelectField } from "~/components/select-field/select-field";
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardContent,
    Button,
    Typography,
    Grid,
    InputLabel
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { User } from "~/models/user";
import { MY_ACCOUNT_VALIDATION} from "~/validators/validators";

type CssRules = 'root' | 'gridItem' | 'label' | 'title' | 'actions';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    gridItem: {
        height: 45,
        marginBottom: 20
    },
    label: {
        fontSize: '0.675rem'
    },
    title: {
        marginBottom: theme.spacing.unit * 3,
        color: theme.palette.grey["600"]
    },
    actions: {
        display: 'flex',
        justifyContent: 'flex-end'
    }
});

export interface MyAccountPanelRootActionProps {}

export interface MyAccountPanelRootDataProps {
    isPristine: boolean;
    isValid: boolean;
    initialValues?: User;
}

const RoleTypes = [
    {key: 'Bio-informatician', value: 'Bio-informatician'},
    {key: 'Data Scientist', value: 'Data Scientist'},
    {key: 'Analyst', value: 'Analyst'},
    {key: 'Researcher', value: 'Researcher'},
    {key: 'Software Developer', value: 'Software Developer'},
    {key: 'System Administrator', value: 'System Administrator'},
    {key: 'Other', value: 'Other'}
];

type MyAccountPanelRootProps = InjectedFormProps<MyAccountPanelRootActionProps> & MyAccountPanelRootDataProps & WithStyles<CssRules>;

export const MyAccountPanelRoot = withStyles(styles)(
    ({ classes, isValid, handleSubmit, reset, isPristine, invalid, submitting }: MyAccountPanelRootProps) => {
        return <Card className={classes.root}>
            <CardContent>
                <Typography variant="title" className={classes.title}>User profile</Typography>
                <form onSubmit={handleSubmit}>
                    <Grid container direction="row" spacing={24}>
                        <Grid item xs={6}>
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="E-mail"
                                    name="email"
                                    component={TextField}
                                    disabled
                                />
                            </Grid>
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="First name"
                                    name="firstName"
                                    component={TextField}
                                    disabled
                                />
                            </Grid>
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="Identity URL"
                                    name="identityUrl"
                                    component={TextField}
                                    disabled
                                />
                            </Grid>
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="Organization"
                                    name="prefs.profile.organization"
                                    component={TextField}
                                    validate={MY_ACCOUNT_VALIDATION}
                                    required
                                />
                            </Grid>
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="Website"
                                    name="prefs.profile.website_url"
                                    component={TextField}
                                />
                            </Grid>
                            <Grid item className={classes.gridItem}>
                                <InputLabel className={classes.label} htmlFor="prefs.profile.role">Organization</InputLabel>
                                <Field
                                    id="prefs.profile.role"
                                    name="prefs.profile.role"
                                    component={NativeSelectField}
                                    items={RoleTypes}
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
                                    disabled
                                />
                            </Grid>
                            <Grid item className={classes.gridItem} />
                            <Grid item className={classes.gridItem}>
                                <Field
                                    label="E-mail at Organization"
                                    name="prefs.profile.organization_email"
                                    component={TextField}
                                    validate={MY_ACCOUNT_VALIDATION}
                                    required
                                />
                            </Grid>
                        </Grid>
                        <Grid item xs={12} className={classes.actions}>
                            <Button color="primary" onClick={reset} disabled={isPristine}>Discard changes</Button>
                            <Button
                                color="primary"
                                variant="contained"
                                type="submit"
                                disabled={isPristine || invalid || submitting}>
                                    Save changes
                            </Button>
                        </Grid>
                    </Grid>
                </form>
            </CardContent>
        </Card>;}
);