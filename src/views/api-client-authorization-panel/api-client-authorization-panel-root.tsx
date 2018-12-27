// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card, CardContent, Grid, Tooltip, IconButton
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { HelpIcon, ShareMeIcon } from '~/components/icon/icon';
import { createTree } from '~/models/tree';
import { DataColumns } from '~/components/data-table/data-table';
import { SortDirection } from '~/components/data-table/data-column';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { API_CLIENT_AUTHORIZATION_PANEL_ID } from '../../store/api-client-authorizations/api-client-authorizations-actions';
import { DataExplorer } from '~/views-components/data-explorer/data-explorer';
import { ResourcesState } from '~/store/resources/resources';
import {
    CommonUuid, TokenApiClientId, TokenApiToken, TokenCreatedByIpAddress, TokenDefaultOwnerUuid, TokenExpiresAt,
    TokenLastUsedAt, TokenLastUsedByIpAddress, TokenScopes, TokenUserId
} from '~/views-components/data-explorer/renderers';

type CssRules = 'card' | 'cardContent' | 'helpIconGrid';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        width: '100%',
        overflow: 'auto'
    },
    cardContent: {
        padding: 0,
        '&:last-child': {
            paddingBottom: 0
        }
    },
    helpIconGrid: {
        textAlign: 'right'
    }
});


export enum ApiClientAuthorizationPanelColumnNames {
    UUID = 'UUID',
    API_CLIENT_ID = 'API Client ID',
    API_TOKEN = 'API Token',
    CREATED_BY_IP_ADDRESS = 'Created by IP address',
    DEFAULT_OWNER_UUID = 'Default owner',
    EXPIRES_AT = 'Expires at',
    LAST_USED_AT = 'Last used at',
    LAST_USED_BY_IP_ADDRESS = 'Last used by IP address',
    SCOPES = 'Scopes',
    USER_ID = 'User ID'
}

export const apiClientAuthorizationPanelColumns: DataColumns<string> = [
    {
        name: ApiClientAuthorizationPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <CommonUuid uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.API_CLIENT_ID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenApiClientId uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.API_TOKEN,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenApiToken uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.CREATED_BY_IP_ADDRESS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenCreatedByIpAddress uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.DEFAULT_OWNER_UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenDefaultOwnerUuid uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.EXPIRES_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenExpiresAt uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.LAST_USED_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenLastUsedAt uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.LAST_USED_BY_IP_ADDRESS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenLastUsedByIpAddress uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.SCOPES,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenScopes uuid={uuid} />
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.USER_ID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <TokenUserId uuid={uuid} />
    }
];

const DEFAULT_MESSAGE = 'Your api client authorization list is empty.';

export interface ApiClientAuthorizationPanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
    openHelpDialog: () => void;
}

export interface ApiClientAuthorizationPanelRootDataProps {
    resources: ResourcesState;
}

type ApiClientAuthorizationPanelRootProps = ApiClientAuthorizationPanelRootActionProps
    & ApiClientAuthorizationPanelRootDataProps & WithStyles<CssRules>;

export const ApiClientAuthorizationPanelRoot = withStyles(styles)(
    ({ classes, onItemDoubleClick, onItemClick, onContextMenu, openHelpDialog }: ApiClientAuthorizationPanelRootProps) =>
        <Card className={classes.card}>
            <CardContent className={classes.cardContent}>
                <Grid container direction="row" justify="flex-end">
                    <Grid item xs={12} className={classes.helpIconGrid}>
                        <Tooltip title="Api token - help">
                            <IconButton onClick={openHelpDialog}>
                                <HelpIcon />
                            </IconButton>
                        </Tooltip>
                    </Grid>
                    <Grid item xs={12}>
                        <DataExplorer
                            id={API_CLIENT_AUTHORIZATION_PANEL_ID}
                            onRowClick={onItemClick}
                            onRowDoubleClick={onItemDoubleClick}
                            onContextMenu={onContextMenu}
                            contextMenuColumn={true}
                            hideColumnSelector
                            hideSearchInput
                            dataTableDefaultView={
                                <DataTableDefaultView
                                    icon={ShareMeIcon}
                                    messages={[DEFAULT_MESSAGE]} />
                            } />
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);