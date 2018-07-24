// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dialog, DialogActions, DialogTitle, DialogContent, WithStyles, withStyles, StyleRulesCallback, Button, Typography, Paper } from '@material-ui/core';
import { ArvadosTheme } from '../../common/custom-theme';

type CssRules = 'link' | 'paper' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        color: theme.palette.primary.main,
        textDecoration: 'none',
        margin: '0px 4px'
    },
    paper: {
        padding: theme.spacing.unit,
        marginBottom: theme.spacing.unit * 2,
        backgroundColor: theme.palette.grey["200"],
        border: `1px solid ${theme.palette.grey["300"]}`
    },
    button: {
        fontSize: '0.8125rem',
        fontWeight: 600
    }
});

interface CurrentTokenDataProps {
    currentToken?: string; 
    open: boolean;
}

interface CurrentTokenActionProps {
    handleClose: () => void;
}

type CurrentTokenProps = CurrentTokenDataProps & CurrentTokenActionProps & WithStyles<CssRules>;

export const CurrentTokenDialog = withStyles(styles)(    
    class extends React.Component<CurrentTokenProps> {
        
        render() {
            const { classes, open, handleClose, currentToken } = this.props;
            return (
                <Dialog open={open} onClose={handleClose} fullWidth={true} maxWidth='md'>
                    <DialogTitle>Current Token</DialogTitle>
                    <DialogContent>
                        <Typography variant='body1' paragraph={true}>
                            The Arvados API token is a secret key that enables the Arvados SDKs to access Arvados with the proper permissions. 
                            <Typography component='p'>
                                For more information see
                                <a href='http://doc.arvados.org/user/reference/api-tokens.html' target='blank' className={classes.link}>
                                    Getting an API token.
                                </a>
                            </Typography>
                        </Typography>

                        <Typography variant='body1' paragraph={true}> 
                            Paste the following lines at a shell prompt to set up the necessary environment for Arvados SDKs to authenticate to your klingenc account.
                        </Typography>

                        <Paper className={classes.paper} elevation={0}>
                            <Typography variant='body1'>
                                HISTIGNORE=$HISTIGNORE:'export ARVADOS_API_TOKEN=*'                            
                            </Typography>
                            <Typography variant='body1'>
                                export ARVADOS_API_TOKEN={currentToken}
                            </Typography>
                            <Typography variant='body1'>
                                export ARVADOS_API_HOST=api.ardev.roche.com
                            </Typography>
                            <Typography variant='body1'>
                                unset ARVADOS_API_HOST_INSECURE
                            </Typography>
                        </Paper>
                        <Typography variant='body1'>
                            Arvados 
                            <a href='http://doc.arvados.org/user/reference/api-tokens.html' target='blank' className={classes.link}>virtual machines</a> 
                            do this for you automatically. This setup is needed only when you use the API remotely (e.g., from your own workstation).
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={handleClose} className={classes.button} color="primary">CLOSE</Button>
                    </DialogActions>
                </Dialog>
            );
        }
    }
);