// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Grid, StyleRulesCallback, Divider, Typography } from '@material-ui/core';
import { Field, WrappedFieldProps } from 'redux-form';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { VisibilityLevelSelect } from './visibility-level-select';
import { VisibilityLevel } from 'store/sharing-dialog/sharing-dialog-types';

const sharingPublicAccessStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit * 2}px 0`,
    }
});

interface AccessProps {
    visibility: VisibilityLevel;
    onSave: () => void;
}

const SharingPublicAccessForm = withStyles(sharingPublicAccessStyles)(
    ({ classes, visibility, onSave }: WithStyles<'root'> & AccessProps) =>
        <>
            <Divider />
            <Grid container alignItems='center' spacing={8} className={classes.root}>
                <Grid item xs={8}>
                    <Typography variant='subtitle1'>
                        {renderVisibilityInfo(visibility)}
                    </Typography>
                </Grid>
                <Grid item xs={4} container wrap='nowrap'>
                    <Field name='visibility' component={VisibilityLevelSelectComponent} onChange={onSave} />
                </Grid>
            </Grid>
        </>
);

const renderVisibilityInfo = (visibility: VisibilityLevel) => {
    switch (visibility) {
        case VisibilityLevel.PUBLIC:
            return 'Anyone can access';
        case VisibilityLevel.SHARED:
            return 'Specific people can access';
        case VisibilityLevel.PRIVATE:
            return 'Only you can access';
        default:
            return '';
    }
};

const SharingPublicAccessFormComponent = ({ visibility, onSave }: AccessProps) =>
    <SharingPublicAccessForm {...{ visibility, onSave }} />;

export default SharingPublicAccessFormComponent;

const VisibilityLevelSelectComponent = ({ input }: WrappedFieldProps) =>
    <VisibilityLevelSelect fullWidth disableUnderline {...input} />;
