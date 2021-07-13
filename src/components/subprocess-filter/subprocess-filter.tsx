// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { Typography, Switch } from '@material-ui/core';

type CssRules = 'container' | 'label' | 'value';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    container: {
        display: 'flex',
        alignItems: 'center',
        height: '20px'
    },
    label: {
        width: '86px',
        color: theme.palette.grey["500"],
        textAlign: 'right',
    },
    value: {
        width: '24px',
        paddingLeft: theme.spacing.unit,
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
        <div className={classes.container} >
            <Typography component="span" className={classes.label}>{label}:</Typography>
            <Typography component="span" className={classes.value}>{value}</Typography>
            {onToggle && <Switch
                checked={checked}
                onChange={onToggle}
                value={key}
                color="primary" />
            }
        </div>
);