// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { 
    StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Grid, Table, 
    TableHead, TableRow, TableCell, TableBody, Tooltip, IconButton 
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { MoreOptionsIcon } from '~/components/icon/icon';
import { NodeResource } from '~/models/node';
import { formatDate } from '~/common/formatters';

type CssRules = 'root' | 'tableRow';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    tableRow: {
        '& th': {
            whiteSpace: 'nowrap'
        }
    }
});

export interface ComputeNodePanelRootActionProps {
    openRowOptions: (event: React.MouseEvent<HTMLElement>, computeNode: NodeResource) => void;
}

export interface ComputeNodePanelRootDataProps {
    computeNodes: NodeResource[];
    hasComputeNodes: boolean;
}

type ComputeNodePanelRootProps = ComputeNodePanelRootActionProps & ComputeNodePanelRootDataProps & WithStyles<CssRules>;

export const ComputeNodePanelRoot = withStyles(styles)(
    ({ classes, hasComputeNodes, computeNodes, openRowOptions }: ComputeNodePanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                {hasComputeNodes && <Grid container direction="row">
                    <Grid item xs={12}>
                        <Table>
                            <TableHead>
                                <TableRow className={classes.tableRow}>
                                    <TableCell>Info</TableCell>
                                    <TableCell>UUID</TableCell>
                                    <TableCell>Domain</TableCell>
                                    <TableCell>First ping at</TableCell>
                                    <TableCell>Hostname</TableCell>
                                    <TableCell>IP Address</TableCell>
                                    <TableCell>Job</TableCell>
                                    <TableCell>Last ping at</TableCell>
                                    <TableCell />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {computeNodes.map((computeNode, index) =>
                                    <TableRow key={index} className={classes.tableRow}>
                                        <TableCell>{JSON.stringify(computeNode.info, null, 4)}</TableCell>
                                        <TableCell>{computeNode.uuid}</TableCell>
                                        <TableCell>{computeNode.domain}</TableCell>
                                        <TableCell>{formatDate(computeNode.firstPingAt) || '(none)'}</TableCell>
                                        <TableCell>{computeNode.hostname || '(none)'}</TableCell>
                                        <TableCell>{computeNode.ipAddress || '(none)'}</TableCell>
                                        <TableCell>{computeNode.jobUuid || '(none)'}</TableCell>
                                        <TableCell>{formatDate(computeNode.lastPingAt) || '(none)'}</TableCell>
                                        <TableCell>
                                            <Tooltip title="More options" disableFocusListener>
                                                <IconButton onClick={event => openRowOptions(event, computeNode)}>
                                                    <MoreOptionsIcon />
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
);