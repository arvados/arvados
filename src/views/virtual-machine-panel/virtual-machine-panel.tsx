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
import { compose } from 'redux';
import { saveRequestedDate, loadVirtualMachinesData } from '~/store/virtual-machines/virtual-machines-actions';
import { RootState } from '~/store/store';
import { ListResults } from '~/services/common-service/common-resource-service';
import { HelpIcon } from '~/components/icon/icon';
import { VirtualMachinesLoginsResource, VirtualMachinesResource } from '~/models/virtual-machines';
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

const mapStateToProps = ({ virtualMachines, auth }: RootState) => {
    return {
        requestedDate: virtualMachines.date,
        isAdmin: auth.user!.isAdmin,
        ...virtualMachines
    };
};

const mapDispatchToProps = {
    saveRequestedDate,
    loadVirtualMachinesData
};

interface VirtualMachinesPanelDataProps {
    requestedDate: string;
    virtualMachines: ListResults<any>;
    logins: VirtualMachinesLoginsResource[];
    links: ListResults<any>;
    isAdmin: boolean;
}

interface VirtualMachinesPanelActionProps {
    saveRequestedDate: () => void;
    loadVirtualMachinesData: () => string;
}

type VirtualMachineProps = VirtualMachinesPanelActionProps & VirtualMachinesPanelDataProps & WithStyles<CssRules>;

export const VirtualMachinePanel = compose(
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
                        {virtualMachines.itemsAvailable === 0 && <CardContentWithNoVirtualMachines {...this.props} />}
                        {virtualMachines.itemsAvailable > 0 && links.itemsAvailable > 0 && <CardContentWithVirtualMachines {...this.props} />}
                        {<CardSSHSection {...this.props} />}
                    </Grid>
                );
            }
        }
    );

const CardContentWithNoVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent className={props.classes.cardWithoutMachines}>
                <Grid item xs={6}>
                    <Typography variant="body2">
                        You do not have access to any virtual machines. Some Arvados features require using the command line. You may request access to a hosted virtual machine with the command line shell.
                    </Typography>
                </Grid>
                <Grid item xs={6} className={props.classes.rightAlign}>
                    <Button variant="contained" color="primary" className={props.classes.button} onClick={props.saveRequestedDate}>
                        SEND REQUEST FOR SHELL ACCESS
                    </Button>
                    {props.requestedDate &&
                        <Typography variant="body1">
                            A request for shell access was sent on {props.requestedDate}
                        </Typography>}
                </Grid>
            </CardContent>
        </Card>
    </Grid>;

const CardContentWithVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                <div className={props.classes.rightAlign}>
                    <Button variant="contained" color="primary" className={props.classes.button} onClick={props.saveRequestedDate}>
                        SEND REQUEST FOR SHELL ACCESS
                    </Button>
                    {props.requestedDate &&
                        <Typography variant="body1">
                            A request for shell access was sent on {props.requestedDate}
                        </Typography>}
                </div>
                <div className={props.classes.icon}>
                    <a href="https://doc.arvados.org/user/getting_started/vm-login-with-webshell.html" target="_blank" className={props.classes.linkIcon}>
                        <Tooltip title="Access VM using webshell">
                            <HelpIcon />
                        </Tooltip>
                    </a>
                </div>
                {console.log(props.isAdmin)}
                {props.isAdmin ? adminVirtualMachinesTable(props) : userVirtualMachinesTable(props)}
            </CardContent>
        </Card>
    </Grid>;

const userVirtualMachinesTable = (props: VirtualMachineProps) =>
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
                    <TableCell>{getUsername(props.links, it)}</TableCell>
                    <TableCell>ssh {getUsername(props.links, it)}@shell.arvados</TableCell>
                    <TableCell>
                        <a href={`https://workbench.c97qk.arvadosapi.com${it.href}/webshell/${getUsername(props.links, it)}`} target="_blank" className={props.classes.link}>
                            Log in as {getUsername(props.links, it)}
                        </a>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;

const adminVirtualMachinesTable = (props: VirtualMachineProps) =>
    <Table>
        <TableHead>
            <TableRow>
                <TableCell>Uuid</TableCell>
                <TableCell>Host name</TableCell>
                <TableCell>Logins</TableCell>
                <TableCell/>
            </TableRow>
        </TableHead>
        <TableBody>
            {props.virtualMachines.items.map((it, index) =>
                <TableRow key={index}>
                    <TableCell>{it.uuid}</TableCell>
                    <TableCell>shell</TableCell>
                    <TableCell>ssh {getUsername(props.links, it)}@shell.arvados</TableCell>
                    <TableCell>
                        <a href={`https://workbench.c97qk.arvadosapi.com${it.href}/webshell/${getUsername(props.links, it)}`} target="_blank" className={props.classes.link}>
                            Log in as {getUsername(props.links, it)}
                        </a>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;

const getUsername = (links: ListResults<any>, virtualMachine: VirtualMachinesResource) => {
    const link = links.items.find((item: any) => item.headUuid === virtualMachine.uuid);
    return link.properties.username || undefined;
};

const CardSSHSection = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                <Typography variant="body2">
                    In order to access virtual machines using SSH, <Link to={Routes.SSH_KEYS} className={props.classes.link}>add an SSH key to your account</Link> and add a section like this to your SSH configuration file ( ~/.ssh/config):
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