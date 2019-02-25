// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { Grid, Typography, Button, Select } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { login, authActions } from '~/store/auth/auth-action';
import { ArvadosTheme } from '~/common/custom-theme';
import { RootState } from '~/store/store';
import * as classNames from 'classnames';

type CssRules = 'root' | 'container' | 'title' | 'content' | 'content__bolder' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        position: 'relative',
        backgroundColor: theme.palette.grey["200"],
        '&::after': {
            content: `''`,
            position: 'absolute',
            top: 0,
            left: 0,
            bottom: 0,
            right: 0,
            background: 'url("arvados-logo-big.png") no-repeat center center',
            opacity: 0.2,
        }
    },
    container: {
        width: '560px',
        zIndex: 10
    },
    title: {
        marginBottom: theme.spacing.unit * 6,
        color: theme.palette.grey["800"]
    },
    content: {
        marginBottom: theme.spacing.unit * 3,
        lineHeight: '1.2rem',
        color: theme.palette.grey["800"]
    },
    'content__bolder': {
        fontWeight: 'bolder'
    },
    button: {
        boxShadow: 'none'
    }
});

type LoginPanelProps = DispatchProp<any> & WithStyles<CssRules> & {
    remoteHosts: { [key: string]: string },
    homeCluster: string,
    uuidPrefix: string
};

export const InactivePanel = withStyles(styles)(
    connect((state: RootState) => ({
        remoteHosts: state.auth.remoteHosts,
        homeCluster: state.auth.homeCluster,
        uuidPrefix: state.auth.localCluster
    }))(({ classes, dispatch, remoteHosts, homeCluster, uuidPrefix }: LoginPanelProps) =>
        <Grid container justify="center" alignItems="center"
            className={classes.root}
            style={{ marginTop: 56, overflowY: "auto", height: "100%" }}>
            <Grid item className={classes.container}>
                <Typography variant='h6' align="center" className={classes.title}>
                    Hi! You're logged in, but...
		</Typography>
                <Typography>
                    Your account is inactive.

		    An administrator must activate your account before you can get any further.
		</Typography>
            </Grid>
        </Grid >
    ));
