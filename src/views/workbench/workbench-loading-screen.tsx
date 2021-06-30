// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { Grid, CircularProgress } from '@material-ui/core';

type CssRules = 'root' | 'img';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    img: {
        marginBottom: theme.spacing.unit * 4
    },
    root: {
        background: theme.palette.background.default,
        bottom: 0,
        left: 0,
        position: 'fixed',
        right: 0,
        top: 0,
        zIndex: theme.zIndex.appBar + 1,
    }
});

export const WorkbenchLoadingScreen = withStyles(styles)(({ classes }: WithStyles<CssRules>) =>
    <Grid container direction="column" alignItems='center' justify='center' className={classes.root}>
        <img src='/arvados_logo.png' className={classes.img} />
        <CircularProgress data-cy='loading-spinner' />
    </Grid>
);
