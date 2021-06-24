// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { useState, useEffect, useRef } from 'react';
import { withStyles, WithStyles, StyleRulesCallback } from '@material-ui/core/styles';
import CircularProgress from '@material-ui/core/CircularProgress';
import { Button, Card, CardContent, TextField, CardActions } from '@material-ui/core';
import { green } from '@material-ui/core/colors';
import { AxiosPromise } from 'axios';
import { DispatchProp } from 'react-redux';
import { saveApiToken } from 'store/auth/auth-action';
import { navigateToRootProject } from 'store/navigation/navigation-action';

type CssRules = 'root' | 'loginBtn' | 'card' | 'wrapper' | 'progress';

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
        display: 'flex',
        flexWrap: 'wrap',
        width: '100%',
        margin: `${theme.spacing.unit} auto`
    },
    loginBtn: {
        marginTop: theme.spacing.unit,
        flexGrow: 1
    },
    card: {
        marginTop: theme.spacing.unit,
        width: '100%'
    },
    wrapper: {
        margin: theme.spacing.unit,
        position: 'relative',
    },
    progress: {
        color: green[500],
        position: 'absolute',
        top: '50%',
        left: '50%',
        marginTop: -12,
        marginLeft: -12,
    },
});

type LoginFormProps = DispatchProp<any> & WithStyles<CssRules> & {
    handleSubmit: (username: string, password: string) => AxiosPromise;
    loginLabel?: string,
};

export const LoginForm = withStyles(styles)(
    ({ handleSubmit, loginLabel, dispatch, classes }: LoginFormProps) => {
        const userInput = useRef<HTMLInputElement>(null);
        const [username, setUsername] = useState('');
        const [password, setPassword] = useState('');
        const [isButtonDisabled, setIsButtonDisabled] = useState(true);
        const [isSubmitting, setSubmitting] = useState(false);
        const [helperText, setHelperText] = useState('');
        const [error, setError] = useState(false);

        useEffect(() => {
            setError(false);
            setHelperText('');
            if (username.trim() && password.trim()) {
                setIsButtonDisabled(false);
            } else {
                setIsButtonDisabled(true);
            }
        }, [username, password]);

        // This only runs once after render.
        useEffect(() => {
            setFocus();
        }, []);

        const setFocus = () => {
            userInput.current!.focus();
        };

        const handleLogin = () => {
            setError(false);
            setHelperText('');
            setSubmitting(true);
            handleSubmit(username, password)
            .then((response) => {
                setSubmitting(false);
                if (response.data.uuid && response.data.api_token) {
                    const apiToken = `v2/${response.data.uuid}/${response.data.api_token}`;
                    dispatch<any>(saveApiToken(apiToken)).finally(
                        () => dispatch(navigateToRootProject));
                } else {
                    setError(true);
                    setHelperText(response.data.message || 'Please try again');
                    setFocus();
                }
            })
            .catch((err) => {
                setError(true);
                setSubmitting(false);
                setHelperText(`${err.response && err.response.data && err.response.data.errors[0] || 'Error logging in: '+err}`);
                setFocus();
            });
        };

        const handleKeyPress = (e: any) => {
            if (e.keyCode === 13 || e.which === 13) {
                if (!isButtonDisabled) {
                    handleLogin();
                }
            }
        };

        return (
            <React.Fragment>
            <form className={classes.root} noValidate autoComplete="off">
                <Card className={classes.card}>
                    <div className={classes.wrapper}>
                    <CardContent>
                        <TextField
                            inputRef={userInput}
                            disabled={isSubmitting}
                            error={error} fullWidth id="username" type="email"
                            label="Username" margin="normal"
                            onChange={(e) => setUsername(e.target.value)}
                            onKeyPress={(e) => handleKeyPress(e)}
                        />
                        <TextField
                            disabled={isSubmitting}
                            error={error} fullWidth id="password" type="password"
                            label="Password" margin="normal"
                            helperText={helperText}
                            onChange={(e) => setPassword(e.target.value)}
                            onKeyPress={(e) => handleKeyPress(e)}
                        />
                    </CardContent>
                    <CardActions>
                        <Button variant="contained" size="large" color="primary"
                            className={classes.loginBtn} onClick={() => handleLogin()}
                            disabled={isSubmitting || isButtonDisabled}>
                            {loginLabel || 'Log in'}
                        </Button>
                    </CardActions>
                    { isSubmitting && <CircularProgress color='secondary' className={classes.progress} />}
                    </div>
                </Card>
            </form>
            </React.Fragment>
        );
    });
