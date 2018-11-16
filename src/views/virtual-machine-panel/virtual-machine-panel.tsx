// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Typography, Button, Card, CardContent } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { DefaultCodeSnippet } from '~/components/default-code-snippet/default-code-snippet';
import { Link } from 'react-router-dom';
import { Dispatch } from 'redux';
import { saveRequestedDate, loadRequestedDate } from '~/store/virtual-machines/virtual-machines-actions';
import { RootState } from '~/store/store';

type CssRules = 'button' | 'codeSnippet' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    button: {
        marginTop: theme.spacing.unit,
        marginBottom: theme.spacing.unit * 2
    },
    codeSnippet: {
        borderRadius: theme.spacing.unit * 0.5,
        border: '1px solid',
        borderColor: theme.palette.grey["400"],
        maxHeight: '400px'
    },
    link: {
        textDecoration: 'none',
        color: theme.palette.primary.main
    },
});

const mapStateToProps = (state: RootState) => {
    return {
        requestedDate: state.virtualMachines.date
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    saveRequestedDate: () => dispatch<any>(saveRequestedDate()),
    loadRequestedDate: () => dispatch<any>(loadRequestedDate())
});

interface VirtualMachinesPanelDataProps {
    requestedDate: string;
}

interface VirtualMachinesPanelActionProps {
    saveRequestedDate: () => void;
    loadRequestedDate: () => string;
}

type VirtualMachineProps = VirtualMachinesPanelActionProps & VirtualMachinesPanelDataProps & WithStyles<CssRules>;

export const VirtualMachinePanel = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    class extends React.Component<VirtualMachineProps> {
        componentDidMount() {
            this.props.loadRequestedDate();
        }

        render() {
            const { classes, saveRequestedDate, requestedDate } = this.props;
            return (
                <Grid container spacing={16}>
                    <Grid item xs={12}>
                        <Card>
                            <CardContent>
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
                            </CardContent>
                        </Card>
                    </Grid>
                    <Grid item xs={12}>
                        <Card>
                            <CardContent>
                                <Typography variant="body2">
                                    In order to access virtual machines using SSH, <Link to='' className={classes.link}>add an SSH key to your account</Link> and add a section like this to your SSH configuration file ( ~/.ssh/config):
                                </Typography>
                                <DefaultCodeSnippet
                                    className={classes.codeSnippet}
                                    lines={[textSSH]} />
                            </CardContent>
                        </Card>
                    </Grid>
                </Grid >
            );
        }
    }));



const textSSH = `Host *.arvados
    TCPKeepAlive yes
    ServerAliveInterval 60
    ProxyCommand ssh -p2222 turnout@switchyard.api.ardev.roche.com -x -a $SSH_PROXY_FLAGS %h`;