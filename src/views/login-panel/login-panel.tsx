// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { Grid, Typography, Button, Select } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { login, authActions } from 'store/auth/auth-action';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { LoginForm } from 'views-components/login-form/login-form';
import Axios from 'axios';
import { Config } from 'common/config';

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

const doPasswordLogin = (url: string) => (username: string, password: string) => {
    const formData = [];
    formData.push('username='+encodeURIComponent(username));
    formData.push('password='+encodeURIComponent(password));
    return Axios.post(`${url}/arvados/v1/users/authenticate`, formData.join('&'), {
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
        },
    });
};

type LoginPanelProps = DispatchProp<any> & WithStyles<CssRules> & {
    remoteHosts: { [key: string]: string },
    homeCluster: string,
    localCluster: string,
    loginCluster: string,
    welcomePage: string,
    passwordLogin: boolean,
};

const loginOptions = ['LDAP', 'PAM', 'Test'];

export const requirePasswordLogin = (config: Config): boolean => {
    if (config && config.clusterConfig && config.clusterConfig.Login) {
        return loginOptions
            .filter(loginOption => !!config.clusterConfig.Login[loginOption])
            .map(loginOption => config.clusterConfig.Login[loginOption].Enable)
            .find(enabled => enabled === true) || false;
    }
    return false;
};

export const LoginPanel = withStyles(styles)(
    connect((state: RootState) => ({
        remoteHosts: state.auth.remoteHosts,
        homeCluster: state.auth.homeCluster,
        localCluster: state.auth.localCluster,
        loginCluster: state.auth.loginCluster,
        welcomePage: state.auth.config.clusterConfig.Workbench.WelcomePageHTML,
        passwordLogin: requirePasswordLogin(state.auth.remoteHostsConfig[state.auth.loginCluster || state.auth.homeCluster]),
        }))(({ classes, dispatch, remoteHosts, homeCluster, localCluster, loginCluster, welcomePage, passwordLogin }: LoginPanelProps) => {
        const loginBtnLabel = `Log in${(localCluster !== homeCluster && loginCluster !== homeCluster) ? " to "+localCluster+" with user from "+homeCluster : ''}`;

        return (<Grid container justify="center" alignItems="center"
            className={classes.root}
            style={{ marginTop: 56, overflowY: "auto", height: "100%" }}>
            <Grid item className={classes.container}>
                <Typography component="div">
                    <div dangerouslySetInnerHTML={{ __html: welcomePage }} style={{ margin: "1em" }} />
                </Typography>
                {Object.keys(remoteHosts).length > 1 && loginCluster === "" &&

                    <Typography component="div" align="right">
                        <label>Please select the cluster that hosts your user account:</label>
                        <Select native value={homeCluster} style={{ margin: "1em" }}
                            onChange={(event) => dispatch(authActions.SET_HOME_CLUSTER(event.target.value))}>
                            {Object.keys(remoteHosts).map((k) => <option key={k} value={k}>{k}</option>)}
                        </Select>
                    </Typography>}

                {passwordLogin
                ? <Typography component="div">
                    <LoginForm dispatch={dispatch}
                        loginLabel={loginBtnLabel}
                        handleSubmit={doPasswordLogin(`https://${remoteHosts[loginCluster || homeCluster]}`)}/>
                </Typography>
                : <Typography component="div" align="right">
                    <Button variant="contained" color="primary" style={{ margin: "1em" }}
                        className={classes.button}
                        onClick={() => dispatch(login(localCluster, homeCluster, loginCluster, remoteHosts))}>
                        {loginBtnLabel}
                    </Button>
                </Typography>}
            </Grid>
        </Grid >);}
    ));
