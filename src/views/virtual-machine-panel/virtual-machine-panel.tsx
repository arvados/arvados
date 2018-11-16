// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Typography, Button, Card, CardContent, TableBody, TableCell, TableHead, TableRow, Table } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { Link } from 'react-router-dom';
import { Dispatch } from 'redux';
import { saveRequestedDate, loadVirtualMachinesData } from '~/store/virtual-machines/virtual-machines-actions';
import { RootState } from '~/store/store';
import { ListResults } from '~/services/common-service/common-resource-service';
import { HelpIcon } from '~/components/icon/icon';
import { VirtualMachinesLoginsResource } from '~/models/virtual-machines';

type CssRules = 'button' | 'codeSnippet' | 'link' | 'linkIcon' | 'icon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        marginTop: theme.spacing.unit,
        marginBottom: theme.spacing.unit * 2
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
        color: theme.palette.grey["400"],
        textAlign: 'right',
        "&:hover": {
            color: theme.palette.common.black,
            transition: 'all 0.5s ease'
        }
    },
    icon: {
        textAlign: "right"
    }
});

const mapStateToProps = (state: RootState) => {
    return {
        requestedDate: state.virtualMachines.date,
        virtualMachines: state.virtualMachines.virtualMachines,
        logins: state.virtualMachines.logins
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    saveRequestedDate: () => dispatch<any>(saveRequestedDate()),
    loadVirtualMachinesData: () => dispatch<any>(loadVirtualMachinesData())
});

interface VirtualMachinesPanelDataProps {
    requestedDate: string;
    virtualMachines: ListResults<any>;
    logins: VirtualMachinesLoginsResource[];
}

interface VirtualMachinesPanelActionProps {
    saveRequestedDate: () => void;
    loadVirtualMachinesData: () => string;
}

type VirtualMachineProps = VirtualMachinesPanelActionProps & VirtualMachinesPanelDataProps & WithStyles<CssRules>;

export const VirtualMachinePanel = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    class extends React.Component<VirtualMachineProps> {
        componentDidMount() {
            this.props.loadVirtualMachinesData();
        }

        render() {
            const { classes, saveRequestedDate, requestedDate, virtualMachines, logins } = this.props;
            return (
                <Grid container spacing={16}>
                    <Grid item xs={12}>
                        <Card>
                            <CardContent>
                                {virtualMachines.itemsAvailable === 0
                                    ? cardContentWithNoVirtualMachines(requestedDate, saveRequestedDate, classes)
                                    : cardContentWithVirtualMachines(virtualMachines, classes)}
                            </CardContent>
                        </Card>
                    </Grid>
                    <Grid item xs={12}>
                        {cardSSHSection(classes)}
                    </Grid>
                </Grid >
            );
        }
    })
);

const cardContentWithNoVirtualMachines = (requestedDate: string, saveRequestedDate: () => void, classes: any) =>
    <span>
        <Typography variant="body2">
            You do not have access to any virtual machines. Some Arvados features require using the command line. You may request access to a hosted virtual machine with the command line shell.
        </Typography>
        <Button variant="contained" color="primary" className={classes.button} onClick={saveRequestedDate}>
            SEND REQUEST FOR SHELL ACCESS
        </Button>
        {requestedDate &&
            <Typography variant="body1">
                A request for shell access was sent on {requestedDate}
            </Typography>}
    </span>;

const login = 'pawelkowalczyk';

const cardContentWithVirtualMachines = (virtualMachines: ListResults<any>, classes: any) =>
    <span>
        <div className={classes.icon}>
            <a href="https://doc.arvados.org/user/getting_started/vm-login-with-webshell.html" target="_blank" className={classes.linkIcon}>
                <HelpIcon />
            </a>
        </div>
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
                {virtualMachines.items.map((it, index) =>
                    <TableRow key={index}>
                        <TableCell>{it.hostname}</TableCell>
                        <TableCell>{login}</TableCell>
                        <TableCell>ssh {login}@shell.arvados</TableCell>
                        <TableCell>
                            <a href={`https://workbench.c97qk.arvadosapi.com${it.href}/webshell/${login}`} target="_blank" className={classes.link}>
                                Log in as {login}
                            </a>
                        </TableCell>
                    </TableRow>
                )}
            </TableBody>
        </Table>
    </span >;

// dodac link do ssh panelu jak juz bedzie
const cardSSHSection = (classes: any) =>
    <Card>
        <CardContent>
            <Typography variant="body2">
                In order to access virtual machines using SSH, <Link to='' className={classes.link}>add an SSH key to your account</Link> and add a section like this to your SSH configuration file ( ~/.ssh/config):
            </Typography>
            <DefaultCodeSnippet
                className={classes.codeSnippet}
                lines={[textSSH]} />
        </CardContent>
    </Card>;

const textSSH = `Host *.arvados
    TCPKeepAlive yes
    ServerAliveInterval 60
    ProxyCommand ssh -p2222 turnout@switchyard.api.ardev.roche.com -x -a $SSH_PROXY_FLAGS %h`;