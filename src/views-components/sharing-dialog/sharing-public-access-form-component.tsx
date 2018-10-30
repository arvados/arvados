// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Divider, Typography } from '@material-ui/core';
import { Field, WrappedFieldProps } from 'redux-form';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { VisibilityLevelSelect } from './visibility-level-select';

const sharingPublicAccessStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit * 2}px 0`,
    }
});

const SharingPublicAccessForm = withStyles(sharingPublicAccessStyles)(
    ({ classes }: WithStyles<'root'>) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} className={classes.root}>
                <Grid item xs={8}>
                    <Typography variant='subheading'>Public visibility</Typography>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field name='visibility' component={VisibilityLevelSelectComponent} />
                </Grid>
            </Grid>
        </>
);

export default () => <SharingPublicAccessForm />;

const VisibilityLevelSelectComponent = ({ input }: WrappedFieldProps) =>
    <VisibilityLevelSelect fullWidth disableUnderline {...input} />;
