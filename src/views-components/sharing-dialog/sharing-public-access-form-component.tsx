// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Divider, Switch, Typography } from '@material-ui/core';
import { Field, WrappedFieldProps, formValues, formValueSelector } from 'redux-form';
import { PermissionSelect, formatPermissionLevel, parsePermissionLevel } from './permission-select';
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
                <Grid item xs={8}>
                    <Typography variant='subheading'>Public access</Typography>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field
                        name='permissions'
                        component={PermissionSelectComponent}
                        format={formatPermissionLevel}
                        parse={parsePermissionLevel}
                    />
                    <Field name='enabled' component={PublicAccessSwitch} />
                </Grid>
            </Grid>
        </>
);

export default () => <SharingPublicAccessForm />;

const PublicAccessSwitch = (props: WrappedFieldProps) =>
    <PublicAccessSwitchComponent {...props} />;

const publicAccessSwitchStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        margin: `0 -7px`,
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
