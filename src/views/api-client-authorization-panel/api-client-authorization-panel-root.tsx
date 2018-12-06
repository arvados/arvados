// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { 
    StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Grid, 
    Table, TableHead, TableRow, TableCell, TableBody, Tooltip, IconButton
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { MoreOptionsIcon } from '~/components/icon/icon';
import { ApiClientAuthorization } from '~/models/api-client-authorization';

type CssRules = 'root' | 'tableRow';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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

export interface ApiClientAuthorizationPanelRootActionProps {
    openRowOptions: (event: React.MouseEvent<HTMLElement>, keepService: ApiClientAuthorization) => void;
}

export interface ApiClientAuthorizationPanelRootDataProps {
    apiClientAuthorizations: ApiClientAuthorization[];
    hasApiClientAuthorizations: boolean;
}

type ApiClientAuthorizationPanelRootProps = ApiClientAuthorizationPanelRootActionProps 
    & ApiClientAuthorizationPanelRootDataProps & WithStyles<CssRules>;

export const ApiClientAuthorizationPanelRoot = withStyles(styles)(
    ({ classes, hasApiClientAuthorizations, apiClientAuthorizations, openRowOptions }: ApiClientAuthorizationPanelRootProps) =>
        <Card className={classes.root}>
            <CardContent>
                {hasApiClientAuthorizations && <Grid container direction="row">
                    <Grid item xs={12}>
                        <Table>
                            <TableHead>
                                <TableRow className={classes.tableRow}>
                                    <TableCell>UUID</TableCell>
                                    <TableCell>API Client ID</TableCell>
                                    <TableCell>API Token</TableCell>
                                    <TableCell>Created by IP address</TableCell>
                                    <TableCell>Default owner</TableCell>
                                    <TableCell>Expires at</TableCell>
                                    <TableCell>Last used at</TableCell>
                                    <TableCell>Last used by IP address</TableCell>
                                    <TableCell>Scopes</TableCell>
                                    <TableCell>User ID</TableCell>
                                    <TableCell />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {apiClientAuthorizations.map((apiClientAuthorizatio, index) =>
                                    <TableRow key={index} className={classes.tableRow}>
                                        <TableCell>{apiClientAuthorizatio.uuid}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.apiClientId}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.apiToken}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.createdByIpAddress || '(none)'}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.defaultOwnerUuid || '(none)'}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.expiresAt || '(none)'}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.lastUsedAt || '(none)'}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.lastUsedByIpAddress || '(none)'}</TableCell>
                                        <TableCell>{JSON.stringify(apiClientAuthorizatio.scopes)}</TableCell>
                                        <TableCell>{apiClientAuthorizatio.userId}</TableCell>
                                        <TableCell>
                                            <Tooltip title="More options" disableFocusListener>
                                                <IconButton onClick={event => openRowOptions(event, apiClientAuthorizatio)}>
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