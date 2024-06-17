// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { Grid, CircularProgress } from '@mui/material';

type CssRules = 'root' | 'img';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    img: {
        marginBottom: theme.spacing(4)
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
    <Grid container direction="column" alignItems='center' justifyContent='center' className={classes.root}>
        <img src='/arvados_logo.png' alt='Arvados logo' className={classes.img} />
        <CircularProgress data-cy='loading-spinner' />
    </Grid>
);
