// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Typography, Button, Card, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Tooltip } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { Link } from 'react-router-dom';
import { compose, Dispatch } from 'redux';
import { saveRequestedDate, loadVirtualMachinesUserData } from '~/store/virtual-machines/virtual-machines-actions';
import { RootState } from '~/store/store';
import { ListResults } from '~/services/common-service/common-service';
import { HelpIcon } from '~/components/icon/icon';
import { Routes } from '~/routes/routes';

type CssRules = 'button' | 'codeSnippet' | 'link' | 'linkIcon' | 'rightAlign' | 'cardWithoutMachines' | 'icon';

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
    }
});

const mapStateToProps = (state: RootState) => {
    return {
        requestedDate: state.virtualMachines.date,
        userUuid: state.auth.user!.uuid,
        ...state.virtualMachines
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<VirtualMachinesPanelActionProps, 'loadVirtualMachinesData' | 'saveRequestedDate'> => ({
    saveRequestedDate: () => dispatch<any>(saveRequestedDate()),
    loadVirtualMachinesData: () => dispatch<any>(loadVirtualMachinesUserData()),
});

interface VirtualMachinesPanelDataProps {
    requestedDate: string;
    virtualMachines: ListResults<any>;
    userUuid: string;
    links: ListResults<any>;
}

interface VirtualMachinesPanelActionProps {
    saveRequestedDate: () => void;
    loadVirtualMachinesData: () => string;
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
                    <Grid container spacing={16}>
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
                        <a href="https://doc.arvados.org/user/getting_started/vm-login-with-webshell.html" target="_blank" className={props.classes.linkIcon}>
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
    <Table>
        <TableHead>
            <TableRow>
                <TableCell>Host name</TableCell>
                <TableCell>Login name</TableCell>
                <TableCell>Command line</TableCell>
                <TableCell>Web shell</TableCell>
            </TableRow>
        </TableHead>
        <TableBody>
            {props.virtualMachines.items.map((it, index) =>
                <TableRow key={index}>
                    <TableCell>{it.hostname}</TableCell>
                    <TableCell>{getUsername(props.links, props.userUuid)}</TableCell>
                    <TableCell>ssh {getUsername(props.links, props.userUuid)}@{it.hostname}.arvados</TableCell>
                    <TableCell>
                        <a href={`https://workbench.c97qk.arvadosapi.com${it.href}/webshell/${getUsername(props.links, props.userUuid)}`} target="_blank" className={props.classes.link}>
                            Log in as {getUsername(props.links, props.userUuid)}
                        </a>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;

const getUsername = (links: ListResults<any>, userUuid: string) => {
    return links.items.map(it => it.tailUuid === userUuid ? it.properties.username : '');
};

const CardSSHSection = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                <Typography variant='body1'>
                    In order to access virtual machines using SSH, <Link to={Routes.SSH_KEYS_USER} className={props.classes.link}>add an SSH key to your account</Link> and add a section like this to your SSH configuration file ( ~/.ssh/config):
                </Typography>
                <DefaultCodeSnippet
                    className={props.classes.codeSnippet}
                    lines={[textSSH]} />
            </CardContent>
        </Card>
    </Grid>;

const textSSH = `Host *.arvados
    TCPKeepAlive yes
    ServerAliveInterval 60
    ProxyCommand ssh -p2222 turnout@switchyard.api.ardev.roche.com -x -a $SSH_PROXY_FLAGS %h`;