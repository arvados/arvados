// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, InjectedFormProps, WrappedFieldProps } from "redux-form";
import { TextField } from "components/text-field/text-field";
import { NativeSelectField } from "components/select-field/select-field";
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
import { ArvadosTheme } from 'common/custom-theme';
import { User } from "models/user";
import { MY_ACCOUNT_VALIDATION } from "validators/validators";

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

export interface MyAccountPanelRootActionProps { }

export interface MyAccountPanelRootDataProps {
    isPristine: boolean;
    isValid: boolean;
    initialValues?: User;
    localCluster: string;
}

const RoleTypes = [
    { key: 'Bio-informatician', value: 'Bio-informatician' },
    { key: 'Data Scientist', value: 'Data Scientist' },
    { key: 'Analyst', value: 'Analyst' },
    { key: 'Researcher', value: 'Researcher' },
    { key: 'Software Developer', value: 'Software Developer' },
    { key: 'System Administrator', value: 'System Administrator' },
    { key: 'Other', value: 'Other' }
];

type MyAccountPanelRootProps = InjectedFormProps<MyAccountPanelRootActionProps> & MyAccountPanelRootDataProps & WithStyles<CssRules>;

type LocalClusterProp = { localCluster: string };
const renderField: React.ComponentType<WrappedFieldProps & LocalClusterProp> = ({ input, localCluster }) => (
    <span>{localCluster === input.value.substring(0, 5) ? "" : "federated"} user {input.value}</span>
);

export const MyAccountPanelRoot = withStyles(styles)(
    ({ classes, isValid, handleSubmit, reset, isPristine, invalid, submitting, localCluster }: MyAccountPanelRootProps) => {
        return <Card className={classes.root}>
            <CardContent>
                <Typography variant="title" className={classes.title}>
                    Logged in as <Field name="uuid" component={renderField} localCluster={localCluster} />
                </Typography>
                <form onSubmit={handleSubmit}>
                    <Grid container spacing={24}>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="First name"
                                name="firstName"
                                component={TextField as any}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="Last name"
                                name="lastName"
                                component={TextField as any}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="E-mail"
                                name="email"
                                component={TextField as any}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="Username"
                                name="username"
                                component={TextField as any}
                                disabled
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="Organization"
                                name="prefs.profile.organization"
                                component={TextField as any}
                                validate={MY_ACCOUNT_VALIDATION}
                                required
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="E-mail at Organization"
                                name="prefs.profile.organization_email"
                                component={TextField as any}
                                validate={MY_ACCOUNT_VALIDATION}
                                required
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <InputLabel className={classes.label} htmlFor="prefs.profile.role">Role</InputLabel>
                            <Field
                                id="prefs.profile.role"
                                name="prefs.profile.role"
                                component={NativeSelectField as any}
                                items={RoleTypes}
                            />
                        </Grid>
                        <Grid item className={classes.gridItem} sm={6} xs={12}>
                            <Field
                                label="Website"
                                name="prefs.profile.website_url"
                                component={TextField as any}
                            />
                        </Grid>
                        <Grid container direction="row" justify="flex-end" >
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
                </form >
            </CardContent >
        </Card >;
    }
);
