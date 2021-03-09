// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    Dialog,
    DialogActions,
    DialogTitle,
    DialogContent,
    WithStyles,
    withStyles,
    StyleRulesCallback,
    Button,
    Typography,
    Tooltip
} from '@material-ui/core';
import * as CopyToClipboard from 'react-copy-to-clipboard';
import { ArvadosTheme } from '~/common/custom-theme';
import { withDialog } from '~/store/dialog/with-dialog';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { connect, DispatchProp } from 'react-redux';
import {
    TokenDialogData,
    getTokenDialogData,
    TOKEN_DIALOG_NAME,
} from '~/store/token-dialog/token-dialog-actions';
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getNewExtraToken } from '~/store/auth/auth-action';
import { CopyIcon } from '~/components/icon/icon';

type CssRules = 'link' | 'paper' | 'button' | 'actionButton' | 'copyIcon' | 'inlineMonospaced';

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
    },
    actionButton: {
        boxShadow: 'none',
        marginTop: theme.spacing.unit * 2,
        marginBottom: theme.spacing.unit * 2,
        marginRight: theme.spacing.unit * 2,
    },
    copyIcon: {
        marginLeft: theme.spacing.unit,
        color: theme.palette.grey["500"],
        cursor: 'pointer',
        display: 'inline',
        '& svg': {
            fontSize: '1rem'
        }
    },
    inlineMonospaced: {
        fontFamily: 'Monospace',
        fontSize: '1rem',
        display: 'inline-block',
    }
});

type TokenDialogProps = TokenDialogData & WithDialogProps<{}> & WithStyles<CssRules> & DispatchProp;

export class TokenDialogComponent extends React.Component<TokenDialogProps> {
    onCopy = (message: string) => {
        this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
            message,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS
        }));
    }

    onGetNewToken = async () => {
        const newToken = await this.props.dispatch<any>(getNewExtraToken());
        if (newToken) {
            this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'New token retrieved',
                hideDuration: 2000,
                kind: SnackbarKind.SUCCESS
            }));
        } else {
            this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                message: 'Creating new tokens is not allowed',
                hideDuration: 2000,
                kind: SnackbarKind.WARNING
            }));
        }
    }

    getSnippet = ({ apiHost, token }: TokenDialogData) =>
        `HISTIGNORE=$HISTIGNORE:'export ARVADOS_API_TOKEN=*'
export ARVADOS_API_TOKEN=${token}
export ARVADOS_API_HOST=${apiHost}
unset ARVADOS_API_HOST_INSECURE`

    render() {
        const { classes, open, closeDialog, ...data } = this.props;

        return <Dialog
            open={open}
            onClose={closeDialog}
            fullWidth={true}
            maxWidth='md'>
            <DialogTitle>Get API Token</DialogTitle>
            <DialogContent>
                <Typography paragraph={true}>
                    The Arvados API token is a secret key that enables the Arvados SDKs to access Arvados with the proper permissions.
                    <Typography component='span'>
                        For more information see
                        <a href='http://doc.arvados.org/user/reference/api-tokens.html' target='blank' className={classes.link}>
                            Getting an API token.
                        </a>
                    </Typography>
                </Typography>
                <Typography  paragraph={true}>
                    <Typography component='span'>
                        Your Arvados API host is: <Typography className={classes.inlineMonospaced} component='span'>{data.apiHost} {<Tooltip title="Copy to clipboard">
                            <span className={classes.copyIcon}>
                                <CopyToClipboard text={data.apiHost} onCopy={() => this.onCopy("API host copied")}>
                                    <CopyIcon />
                                </CopyToClipboard>
                            </span>
                        </Tooltip>}
                        </Typography>
                    </Typography>
                    <Typography component='span'>
                        Your token is: <Typography className={classes.inlineMonospaced} component='span'>{data.token} {<Tooltip title="Copy to clipboard">
                            <span className={classes.copyIcon}>
                                <CopyToClipboard text={data.token} onCopy={() => this.onCopy("Token copied")}>
                                    <CopyIcon />
                                </CopyToClipboard>
                            </span>
                            </Tooltip>}
                        </Typography>
                    </Typography>
                    <Typography component='span'>
                    { data.tokenExpiration
                        ? `Expires at: ${data.tokenExpiration.toLocaleString()}`
                        : `This token does not have an expiration date`
                    }
                    </Typography>
                </Typography>
                <Typography paragraph={true}>
                    Paste the following lines at a shell prompt to set up the necessary environment for Arvados SDKs to authenticate to your account.
                </Typography>
                <DefaultCodeSnippet lines={[this.getSnippet(data)]} />
                <CopyToClipboard text={this.getSnippet(data)} onCopy={() => this.onCopy('Shell code block copied')}>
                    <Button
                        color="primary"
                        size="small"
                        variant="contained"
                        className={classes.actionButton}
                    >
                        COPY TO CLIPBOARD
                    </Button>
                </CopyToClipboard>
                { this.props.canCreateNewTokens && <Button
                    onClick={() => this.onGetNewToken()}
                    color="primary"
                    size="small"
                    variant="contained"
                    className={classes.actionButton}
                >
                    GET NEW TOKEN
                </Button> }
                <Typography >
                    Arvados
                            <a href='http://doc.arvados.org/user/reference/api-tokens.html' target='blank' className={classes.link}>virtual machines</a>
                    do this for you automatically. This setup is needed only when you use the API remotely (e.g., from your own workstation).
                </Typography>
            </DialogContent>
            <DialogActions>
                <Button onClick={closeDialog} className={classes.button} color="primary">CLOSE</Button>
            </DialogActions>
        </Dialog>;
    }
}

export const TokenDialog =
    withStyles(styles)(
        connect(getTokenDialogData)(
            withDialog(TOKEN_DIALOG_NAME)(TokenDialogComponent)));

