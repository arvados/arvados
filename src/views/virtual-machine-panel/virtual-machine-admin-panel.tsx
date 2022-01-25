// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Grid, Card, Chip, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Tooltip, IconButton } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { compose, Dispatch } from 'redux';
import { loadVirtualMachinesAdminData, openAddVirtualMachineLoginDialog, openRemoveVirtualMachineLoginDialog } from 'store/virtual-machines/virtual-machines-actions';
import { RootState } from 'store/store';
import { ListResults } from 'services/common-service/common-service';
import { MoreOptionsIcon, AddUserIcon } from 'components/icon/icon';
import { VirtualMachineLogins, VirtualMachinesResource } from 'models/virtual-machines';
import { openVirtualMachinesContextMenu } from 'store/context-menu/context-menu-actions';
import { ResourceUuid, VirtualMachineHostname, VirtualMachineLogin } from 'views-components/data-explorer/renderers';

type CssRules = 'moreOptionsButton' | 'moreOptions' | 'chipsRoot';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    moreOptionsButton: {
        padding: 0
    },
    moreOptions: {
        textAlign: 'right',
        '&:last-child': {
            paddingRight: 0
        }
    },
    chipsRoot: {
        margin: `0px -${theme.spacing.unit / 2}px`,
    },
});

const mapStateToProps = (state: RootState) => {
    return {
        userUuid: state.auth.user!.uuid,
        ...state.virtualMachines
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<VirtualMachinesPanelActionProps, 'loadVirtualMachinesData' | 'onOptionsMenuOpen' | 'onAddLogin' | 'onDeleteLogin'> => ({
    loadVirtualMachinesData: () => dispatch<any>(loadVirtualMachinesAdminData()),
    onOptionsMenuOpen: (event, virtualMachine) => {
        dispatch<any>(openVirtualMachinesContextMenu(event, virtualMachine));
    },
    onAddLogin: (uuid: string) => {
        dispatch<any>(openAddVirtualMachineLoginDialog(uuid));
    },
    onDeleteLogin: (uuid: string) => {
        dispatch<any>(openRemoveVirtualMachineLoginDialog(uuid));
    },
});

interface VirtualMachinesPanelDataProps {
    virtualMachines: ListResults<any>;
    logins: VirtualMachineLogins;
    links: ListResults<any>;
    userUuid: string;
}

interface VirtualMachinesPanelActionProps {
    loadVirtualMachinesData: () => string;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, virtualMachine: VirtualMachinesResource) => void;
    onAddLogin: (uuid: string) => void;
    onDeleteLogin: (uuid: string) => void;
}

type VirtualMachineProps = VirtualMachinesPanelActionProps & VirtualMachinesPanelDataProps & WithStyles<CssRules>;

export const VirtualMachineAdminPanel = compose(
    withStyles(styles),
    connect(mapStateToProps, mapDispatchToProps))(
        class extends React.Component<VirtualMachineProps> {
            componentDidMount() {
                this.props.loadVirtualMachinesData();
            }

            render() {
                const { virtualMachines } = this.props;
                return (
                    <Grid container spacing={16}>
                        {virtualMachines.itemsAvailable > 0 && <CardContentWithVirtualMachines {...this.props} />}
                    </Grid>
                );
            }
        }
    );

const CardContentWithVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent>
                {virtualMachinesTable(props)}
            </CardContent>
        </Card>
    </Grid>;

const virtualMachinesTable = (props: VirtualMachineProps) =>
    <Table>
        <TableHead>
            <TableRow>
                <TableCell>Uuid</TableCell>
                <TableCell>Host name</TableCell>
                <TableCell>Logins</TableCell>
                <TableCell />
                <TableCell />
            </TableRow>
        </TableHead>
        <TableBody>
            {props.logins.items.length > 0 && props.virtualMachines.items.map((machine, index) =>
                <TableRow key={index}>
                    <TableCell><ResourceUuid uuid={machine.uuid} /></TableCell>
                    <TableCell><VirtualMachineHostname uuid={machine.uuid} /></TableCell>
                    <TableCell>
                        <Grid container spacing={8} className={props.classes.chipsRoot}>
                            {props.links.items.filter((link) => (link.headUuid === machine.uuid)).map((permission, i) => (
                                <Grid item key={i}>
                                    <Chip label={<VirtualMachineLogin linkUuid={permission.uuid} />} onDelete={event => props.onDeleteLogin(permission.uuid)} />
                                </Grid>
                            ))}
                        </Grid>
                    </TableCell>
                    <TableCell>
                        <Tooltip title="Add Login Permission" disableFocusListener>
                            <IconButton onClick={event => props.onAddLogin(machine.uuid)} className={props.classes.moreOptionsButton}>
                                <AddUserIcon />
                            </IconButton>
                        </Tooltip>
                    </TableCell>
                    <TableCell className={props.classes.moreOptions}>
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton onClick={event => props.onOptionsMenuOpen(event, machine)} className={props.classes.moreOptionsButton}>
                                <MoreOptionsIcon />
                            </IconButton>
                        </Tooltip>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;
