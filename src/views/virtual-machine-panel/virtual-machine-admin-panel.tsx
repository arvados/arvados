// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { connect } from 'react-redux';
import { Grid, Card, CardContent, TableBody, TableCell, TableHead, TableRow, Table, Tooltip, IconButton } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose, Dispatch } from 'redux';
import { loadVirtualMachinesAdminData } from '~/store/virtual-machines/virtual-machines-actions';
import { RootState } from '~/store/store';
import { ListResults } from '~/services/common-service/common-service';
import { MoreOptionsIcon } from '~/components/icon/icon';
import { VirtualMachineLogins, VirtualMachinesResource } from '~/models/virtual-machines';
import { openVirtualMachinesContextMenu } from '~/store/context-menu/context-menu-actions';

type CssRules = 'moreOptionsButton' | 'moreOptions';

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
});

const mapStateToProps = (state: RootState) => {
    return {
        logins: state.virtualMachines.logins,
        userUuid: state.auth.user!.uuid,
        ...state.virtualMachines
    };
};

const mapDispatchToProps = (dispatch: Dispatch): Pick<VirtualMachinesPanelActionProps, 'loadVirtualMachinesData' | 'onOptionsMenuOpen'> => ({
    loadVirtualMachinesData: () => dispatch<any>(loadVirtualMachinesAdminData()),
    onOptionsMenuOpen: (event, virtualMachine) => {
        dispatch<any>(openVirtualMachinesContextMenu(event, virtualMachine));
    },
});

interface VirtualMachinesPanelDataProps {
    virtualMachines: ListResults<any>;
    logins: VirtualMachineLogins;
    userUuid: string;
}

interface VirtualMachinesPanelActionProps {
    loadVirtualMachinesData: () => string;
    onOptionsMenuOpen: (event: React.MouseEvent<HTMLElement>, virtualMachine: VirtualMachinesResource) => void;
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
            </TableRow>
        </TableHead>
        <TableBody>
            {props.logins.items.length > 0 && props.virtualMachines.items.map((it, index) =>
                <TableRow key={index}>
                    <TableCell>{it.uuid}</TableCell>
                    <TableCell>{it.hostname}</TableCell>
                    <TableCell>["{props.logins.items.map(it => it.userUuid === props.userUuid ? it.username : '')}"]</TableCell>
                    <TableCell className={props.classes.moreOptions}>
                        <Tooltip title="More options" disableFocusListener>
                            <IconButton onClick={event => props.onOptionsMenuOpen(event, it)} className={props.classes.moreOptionsButton}>
                                <MoreOptionsIcon />
                            </IconButton>
                        </Tooltip>
                    </TableCell>
                </TableRow>
            )}
        </TableBody>
    </Table>;
