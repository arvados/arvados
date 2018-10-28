// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Divider, Switch } from '@material-ui/core';
import { Field, WrappedFieldProps, formValues, formValueSelector } from 'redux-form';
import { PermissionSelect } from './permission-select';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { connect } from 'react-redux';

const sharingPublicAccessStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    }
});

const SharingPublicAccessForm = withStyles(sharingPublicAccessStyles)(
    ({ classes }: WithStyles<'root'>) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} className={classes.root}>
                <Grid item xs={6}>
                    Public access
                </Grid>
                <Grid item xs={2}>
                    <Field name='enabled' component={PublicAccessSwitch} />
                </Grid>
                <Grid item xs={4}>
                    <Field name='permissions' component={PermissionSelectComponent} />
                </Grid>
            </Grid>
        </>
);

export default () => <SharingPublicAccessForm />;

const PublicAccessSwitch = (props: WrappedFieldProps) =>
    <PublicAccessSwitchComponent {...props} />;

const publicAccessSwitchStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        margin: `-${theme.spacing.unit * 2}px auto`,
    }
});

const PublicAccessSwitchComponent = withStyles(publicAccessSwitchStyles)(
    ({ input, classes }: WrappedFieldProps & WithStyles<'root'>) =>
        <Switch checked={input.value} onChange={input.onChange} color='primary' classes={classes} />
);

const PermissionSelectComponent = connect(
    (state: any, props: WrappedFieldProps) => ({
        disabled: !formValueSelector(props.meta.form)(state, 'enabled'),
    })
)(({ input, disabled }: WrappedFieldProps & { disabled: boolean }) => {
    return <PermissionSelect disabled={disabled} fullWidth disableUnderline {...input} />;
});
