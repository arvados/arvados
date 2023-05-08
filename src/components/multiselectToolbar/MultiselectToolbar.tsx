// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'root' | 'item';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        // border: '2px dotted green',
        display: 'flex',
        flexDirection: 'row',
    },
    item: {
        // border: '2px dotted blue',
        color: theme.palette.text.primary,
        margin: '0.5rem',
    },
});

type MultiselectToolbarProps = WithStyles<CssRules>;

export default withStyles(styles)((props: MultiselectToolbarProps) => {
    console.log(props);
    const { classes } = props;
    return (
        <Toolbar className={classes.root}>
            <div className={classes.item}>test1</div>
            <div className={classes.item}>test2</div>
            <div className={classes.item}>test3</div>
        </Toolbar>
    );
});
