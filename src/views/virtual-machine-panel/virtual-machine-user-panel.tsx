// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Grid, Typography, Button, Card, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Tooltip, Chip } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { compose, Dispatch } from 'redux';
import { saveRequestedDate, loadVirtualMachinesUserData } from 'store/virtual-machines/virtual-machines-actions';
import { RootState } from 'store/store';
import { ListResults } from 'services/common-service/common-service';
import { HelpIcon } from 'components/icon/icon';
import { SESSION_STORAGE } from "services/auth-service/auth-service";
// import * as CopyToClipboard from 'react-copy-to-clipboard';
import parse from "parse-duration";
import { CopyIcon } from 'components/icon/icon';
import CopyToClipboard from 'react-copy-to-clipboard';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';

type CssRules = 'button' | 'codeSnippet' | 'link' | 'linkIcon' | 'rightAlign' | 'cardWithoutMachines' | 'icon' | 'chipsRoot' | 'copyIcon' | 'webshellButton';

const EXTRA_TOKEN = "exraToken";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        marginTop: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    },
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: '1px solid',
        borderColor: theme.palette.grey["400"],
    },
    link: {
        textDecoration: 'none',
        color: theme.palette.primary.main,
        "&:hover": {
            color: theme.palette.primary.dark,
            transition: 'all 0.5s ease'
        }
    },
    linkIcon: {
        textDecoration: 'none',
        color: theme.palette.grey["500"],
        textAlign: 'right',
        "&:hover": {
            color: theme.palette.common.black,
            transition: 'all 0.5s ease'
        }
    },
    rightAlign: {
        textAlign: "right"
    },
    cardWithoutMachines: {
        display: 'flex'
    },
    icon: {
        textAlign: "right",
        marginTop: theme.spacing.unit
    },
    chipsRoot: {
        margin: `0px -${theme.spacing.unit / 2}px`,
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
    webshellButton: {
        textTransform: "initial",
    },
});

const mapStateToProps = (state: RootState) => {
    return {
        requestedDate: state.virtualMachines.date,
        userUuid: state.auth.user!.uuid,
        helpText: state.auth.config.clusterConfig.Workbench.SSHHelpPageHTML,
        hostSuffix: state.auth.config.clusterConfig.Workbench.SSHHelpHostSuffix || "",
        token: state.auth.extraApiToken || state.auth.apiToken || '',
        tokenLocation: state.auth.extraApiToken ? EXTRA_TOKEN : (state.auth.apiTokenLocation || ''),
        webshellUrl: state.auth.config.clusterConfig.Services.WebShell.ExternalURL,
        idleTimeout: parse(state.auth.config.clusterConfig.Workbench.IdleTimeout, 's') || 0,
        ...state.virtualMachines
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<VirtualMachinesPanelActionProps, 'loadVirtualMachinesData' | 'saveRequestedDate' | 'onCopy'> => ({
    saveRequestedDate: () => dispatch<any>(saveRequestedDate()),
    loadVirtualMachinesData: () => dispatch<any>(loadVirtualMachinesUserData()),
    onCopy: (message: string) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message,
            hideDuration: 2000,
            kind: SnackbarKind.SUCCESS
        }));
    },
});

interface VirtualMachinesPanelDataProps {
    requestedDate: string;
    virtualMachines: ListResults<any>;
    userUuid: string;
    links: ListResults<any>;
    helpText: string;
    hostSuffix: string;
    token: string;
    tokenLocation: string;
    webshellUrl: string;
    idleTimeout: number;
}

interface VirtualMachinesPanelActionProps {
    saveRequestedDate: () => void;
    loadVirtualMachinesData: () => string;
    onCopy: (message: string) => void;
}

type VirtualMachineProps = VirtualMachinesPanelActionProps & VirtualMachinesPanelDataProps & WithStyles<CssRules>;

export const VirtualMachineUserPanel = compose(
    withStyles(styles),
    connect(mapStateToProps, mapDispatchToProps))(
        class extends React.Component<VirtualMachineProps> {
            componentDidMount() {
                this.props.loadVirtualMachinesData();
            }

            render() {
                const { virtualMachines, links } = this.props;
                return (
                    <Grid container spacing={16} data-cy="vm-user-panel">
                        {virtualMachines.itemsAvailable === 0 && <CardContentWithoutVirtualMachines {...this.props} />}
                        {virtualMachines.itemsAvailable > 0 && links.itemsAvailable > 0 && <CardContentWithVirtualMachines {...this.props} />}
                        {<CardSSHSection {...this.props} />}
                    </Grid>
                );
            }
        }
    );

const CardContentWithoutVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent className={props.classes.cardWithoutMachines}>
                <Grid item xs={6}>
                    <Typography variant='body1'>
                        You do not have access to any virtual machines. Some Arvados features require using the command line. You may request access to a hosted virtual machine with the command line shell.
                    </Typography>
                </Grid>
                <Grid item xs={6} className={props.classes.rightAlign}>
                    {virtualMachineSendRequest(props)}
                </Grid>
            </CardContent>
        </Card>
    </Grid>;

const CardContentWithVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                <span>
                    <div className={props.classes.rightAlign}>
                        {virtualMachineSendRequest(props)}
                    </div>
                    <div className={props.classes.icon}>
                        <a href="https://doc.arvados.org/user/getting_started/vm-login-with-webshell.html" target="_blank" rel="noopener noreferrer" className={props.classes.linkIcon}>
                            <Tooltip title="Access VM using webshell">
                                <HelpIcon />
                            </Tooltip>
                        </a>
                    </div>
                    {virtualMachinesTable(props)}
                </span>

            </CardContent>
        </Card>
    </Grid>;

const virtualMachineSendRequest = (props: VirtualMachineProps) =>
    <span>
        <Button variant="contained" color="primary" className={props.classes.button} onClick={props.saveRequestedDate}>
            SEND REQUEST FOR SHELL ACCESS
        </Button>
        {props.requestedDate &&
            <Typography >
                A request for shell access was sent on {props.requestedDate}
            </Typography>}
    </span>;

const virtualMachinesTable = (props: VirtualMachineProps) =>
    <Table data-cy="vm-user-table">
        <TableHead>
            <TableRow>
                <TableCell>Host name</TableCell>
                <TableCell>Login name</TableCell>
                <TableCell>Groups</TableCell>
                <TableCell>Command line</TableCell>
                <TableCell>Web shell</TableCell>
            </TableRow>
        </TableHead>
        <TableBody>
            {props.virtualMachines.items.map(it =>
                props.links.items.map(lk => {
                    if (lk.tailUuid === props.userUuid && lk.headUuid === it.uuid) {
                        const username = lk.properties.username;
                        const command = `ssh ${username}@${it.hostname}${props.hostSuffix}`;
                        let tokenParam = "";
                        if (props.tokenLocation === SESSION_STORAGE || props.tokenLocation === EXTRA_TOKEN) {
                          tokenParam = `&token=${encodeURIComponent(props.token)}`;
                        }
                        const loginHref = `/webshell/?host=${encodeURIComponent(props.webshellUrl + '/' + it.hostname)}&timeout=${props.idleTimeout}&login=${encodeURIComponent(username)}${tokenParam}`;
                        return <TableRow key={lk.uuid}>
                            <TableCell>{it.hostname}</TableCell>
                            <TableCell>{username}</TableCell>
                            <TableCell>
                                <Grid container spacing={8} className={props.classes.chipsRoot}>
                                    {
                                    (lk.properties.groups || []).map((group, i) => (
                                        <Grid item key={i}>
                                            <Chip label={group} />
                                        </Grid>
                                    ))
                                    }
                                </Grid>
                            </TableCell>
                            <TableCell>
                                {command}
                                <Tooltip title="Copy to clipboard">
                                    <span className={props.classes.copyIcon}>
                                        <CopyToClipboard text={command || ""} onCopy={() => props.onCopy!("Copied")}>
                                            <CopyIcon />
                                        </CopyToClipboard>
                                    </span>
                                </Tooltip>
                            </TableCell>
                            <TableCell>
                                <Button
                                    className={props.classes.webshellButton}
                                    variant="contained"
                                    size="small"
                                    href={loginHref}
                                    target="_blank"
                                    rel="noopener noreferrer">
                                        Log in as {username}
                                </Button>
                            </TableCell>
                        </TableRow>;
                    }
                    return null;
                }
                ))}
        </TableBody>
    </Table>;

const CardSSHSection = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                <Typography>
                    <div dangerouslySetInnerHTML={{ __html: props.helpText }} style={{ margin: "1em" }} />
                </Typography>
            </CardContent>
        </Card>
    </Grid>;
