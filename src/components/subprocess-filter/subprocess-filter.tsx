// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { Grid, Typography, Switch } from '@material-ui/core';

type CssRules = 'grid' | 'label' | 'value' | 'switch';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    grid: {
        display: 'flex'
    },
    label: {
        width: '86px',
        color: theme.palette.grey["500"],
        textAlign: 'right'
    },
    value: {
        width: '24px',
        paddingLeft: theme.spacing.unit
    },
    switch: {
        '& span:first-child': {
            height: '18px'
        }
    }
});

export interface SubprocessFilterDataProps {
    label: string;
    value: number;
    checked?: boolean;
    key?: string;
    onToggle?: () => void;
}

type SubprocessFilterProps = SubprocessFilterDataProps & WithStyles<CssRules>;

export const SubprocessFilter = withStyles(styles)(
    ({ classes, label, value, key, checked, onToggle }: SubprocessFilterProps) =>
        <Grid item className={classes.grid} md={12} lg={6} >
            <Typography component="span" className={classes.label}>{label}:</Typography>
            <Typography component="span" className={classes.value}>{value}</Typography>
            {onToggle && <Switch classes={{ root: classes.switch }}
                checked={checked}
                onChange={onToggle}
                value={key}
                color="primary" />
            }
        </Grid>
);