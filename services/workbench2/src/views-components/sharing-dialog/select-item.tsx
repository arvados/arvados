// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Grid, withStyles, StyleRulesCallback } from '@material-ui/core';
import { WithStyles } from '@material-ui/core/styles';
import { SvgIconProps } from '@material-ui/core/SvgIcon';

type SelectItemClasses = 'value' | 'icon';

const permissionItemStyles: StyleRulesCallback<SelectItemClasses> = theme => ({
    value: {
        marginLeft: theme.spacing.unit,
    },
    icon: {
        margin: `-${theme.spacing.unit / 2}px 0`
    }
});

/**
 * This component should be used as a child of MenuItem component.
 */
export const SelectItem = withStyles(permissionItemStyles)(
    ({ value, icon: Icon, classes }: { value: string, icon: React.ComponentType<SvgIconProps> } & WithStyles<SelectItemClasses>) => {
        return (
            <Grid container alignItems='center'>
                <Icon className={classes.icon} />
                <span className={classes.value}>
                    {value}
                </span>
            </Grid>);
    });

