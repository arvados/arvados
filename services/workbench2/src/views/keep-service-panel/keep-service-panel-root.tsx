// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    Card,
    CardContent,
    Grid,
    Table,
    TableHead,
    TableRow,
    TableCell,
    TableBody,
    Tooltip,
    IconButton,
    Checkbox,
} from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { MoreVerticalIcon } from 'components/icon/icon';
import { KeepServiceResource } from 'models/keep-services';

type CssRules = 'root' | 'tableRow';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    tableRow: {
        '& td, th': {
            whiteSpace: 'nowrap'
        }
    }
});

export interface KeepServicePanelRootActionProps {
    openRowOptions: (event: React.MouseEvent<HTMLElement>, keepService: KeepServiceResource) => void;
}

export interface KeepServicePanelRootDataProps {
    keepServices: KeepServiceResource[];
}

type KeepServicePanelRootProps = KeepServicePanelRootActionProps & KeepServicePanelRootDataProps & WithStyles<CssRules>;

export const KeepServicePanelRoot = withStyles(styles)(
    ({ classes, keepServices, openRowOptions }: KeepServicePanelRootProps) => {
        const hasKeepSerices = keepServices.length > 0;
        return <Card className={classes.root}>
            <CardContent>
                {hasKeepSerices && <Grid container direction="row">
                    <Grid item xs={12}>
                        <Table>
                            <TableHead>
                                <TableRow className={classes.tableRow}>
                                    <TableCell>UUID</TableCell>
                                    <TableCell>Read only</TableCell>
                                    <TableCell>Service host</TableCell>
                                    <TableCell>Service port</TableCell>
                                    <TableCell>Service SSL flag</TableCell>
                                    <TableCell>Service type</TableCell>
                                    <TableCell />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {keepServices.map((keepService, index) =>
                                    <TableRow key={index} className={classes.tableRow}>
                                        <TableCell>{keepService.uuid}</TableCell>
                                        <TableCell>
                                            <Checkbox
                                                disableRipple
                                                color="primary"
                                                checked={keepService.readOnly} />
                                        </TableCell>
                                        <TableCell>{keepService.serviceHost}</TableCell>
                                        <TableCell>{keepService.servicePort}</TableCell>
                                        <TableCell>
                                            <Checkbox
                                                disableRipple
                                                color="primary"
                                                checked={keepService.serviceSslFlag} />
                                        </TableCell>
                                        <TableCell>{keepService.serviceType}</TableCell>
                                        <TableCell>
                                            <Tooltip title="More options" disableFocusListener>
                                                <IconButton onClick={event => openRowOptions(event, keepService)} size="large">
                                                    <MoreVerticalIcon />
                                                </IconButton>
                                            </Tooltip>
                                        </TableCell>
                                    </TableRow>)}
                            </TableBody>
                        </Table>
                    </Grid>
                </Grid>}
            </CardContent>
        </Card>
    }
);
