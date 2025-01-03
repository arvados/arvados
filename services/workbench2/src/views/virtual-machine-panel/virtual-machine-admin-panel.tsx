// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Grid, Card, Chip, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Typography, Tooltip, IconButton } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { compose, Dispatch } from 'redux';
import { loadVirtualMachinesAdminData, openAddVirtualMachineLoginDialog, openRemoveVirtualMachineLoginDialog, openEditVirtualMachineLoginDialog } from 'store/virtual-machines/virtual-machines-actions';
import { RootState } from 'store/store';
import { ListResults } from 'services/common-service/common-service';
import { MoreVerticalIcon, AddUserIcon } from 'components/icon/icon';
import { VirtualMachineLogins, VirtualMachinesResource } from 'models/virtual-machines';
import { openVirtualMachinesContextMenu } from 'store/context-menu/context-menu-actions';
import { VirtualMachineHostname, VirtualMachineLogin } from 'views-components/data-explorer/renderers';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';

type CssRules = 'moreOptionsButton' | 'moreOptions' | 'chipsRoot' | 'vmTableWrapper';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
        margin: `0px -${theme.spacing(0.5)}`,
    },
    vmTableWrapper: {
        overflowX: 'auto',
    },
});

const mapStateToProps = (state: RootState) => {
    return {
        userUuid: state.auth.user!.uuid,
        ...state.virtualMachines
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<VirtualMachinesPanelActionProps, 'loadVirtualMachinesData' | 'onOptionsMenuOpen' | 'onAddLogin' | 'onDeleteLogin' | 'onLoginEdit'> => ({
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
    onLoginEdit: (uuid: string) => {
        dispatch<any>(openEditVirtualMachineLoginDialog(uuid));
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
    onLoginEdit: (uuid: string) => void;
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
                    <Grid container spacing={2}>
                        {virtualMachines.items.length > 0 && <CardContentWithVirtualMachines {...this.props} />}
                    </Grid>
                );
            }
        }
    );

const CardContentWithVirtualMachines = (props: VirtualMachineProps) =>
    <Grid item xs={12}>
        <Card>
            <CardContent className={props.classes.vmTableWrapper}>
                {virtualMachinesTable(props)}
            </CardContent>
        </Card>
    </Grid>;

const virtualMachinesTable = (props: VirtualMachineProps) =>
    <Table data-cy="vm-admin-table">
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
            {props.virtualMachines.items.map((machine, index) =>
                <TableRow key={index}>
                    <TableCell>
                        <Typography
                            data-cy="uuid"
                            noWrap
                        >
                            {machine.uuid}
                            {(machine.uuid && <CopyToClipboardSnackbar value={machine.uuid} />) || "-"}
                        </Typography>
                    </TableCell>
                    <TableCell><VirtualMachineHostname uuid={machine.uuid} /></TableCell>
                    <TableCell>
                        <Grid container spacing={1} className={props.classes.chipsRoot}>
                            {props.links.items.filter((link) => (link.headUuid === machine.uuid)).map((permission, i) => (
                                <Grid item key={i}>
                                    <Chip label={<VirtualMachineLogin linkUuid={permission.uuid} />} onDelete={event => props.onDeleteLogin(permission.uuid)} onClick={event => props.onLoginEdit(permission.uuid)} />
                                </Grid>
                            ))}
                        </Grid>
                    </TableCell>
                    <TableCell>
                        <Tooltip title="Add Login Permission" disableFocusListener>
                            <IconButton
                                onClick={event => props.onAddLogin(machine.uuid)}
                                className={props.classes.moreOptionsButton}
                                size="large">
                                <AddUserIcon />
                            </IconButton>
                        </Tooltip>
                    </TableCell>
                    <TableCell className={props.classes.moreOptions}>
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton
                                onClick={event => props.onOptionsMenuOpen(event, machine)}
                                className={props.classes.moreOptionsButton}
                                size="large">
                                <MoreVerticalIcon />
                            </IconButton>
                        </Tooltip>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;
