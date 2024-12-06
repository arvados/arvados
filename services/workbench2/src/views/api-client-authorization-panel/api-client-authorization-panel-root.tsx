// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { ShareMeIcon } from 'components/icon/icon';
import { createTree } from 'models/tree';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { API_CLIENT_AUTHORIZATION_PANEL_ID } from '../../store/api-client-authorizations/api-client-authorizations-actions';
import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { ResourcesState } from 'store/resources/resources';
import {
    renderUuid,
    renderString,
    renderDate,
} from 'views-components/data-explorer/renderers';
import { ApiClientAuthorization } from 'models/api-client-authorization';

type CssRules = 'root';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    }
});


export enum ApiClientAuthorizationPanelColumnNames {
    UUID = 'UUID',
    API_TOKEN = 'API Token',
    CREATED_BY_IP_ADDRESS = 'Created by IP address',
    EXPIRES_AT = 'Expires at',
    LAST_USED_AT = 'Last used at',
    LAST_USED_BY_IP_ADDRESS = 'Last used by IP address',
    SCOPES = 'Scopes',
    USER_ID = 'User ID'
}

export const apiClientAuthorizationPanelColumns: DataColumns<ApiClientAuthorization> = [
    {
        name: ApiClientAuthorizationPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "uuid"},
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderUuid({uuid: resource.uuid})
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.API_TOKEN,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderString(resource.apiToken)
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.CREATED_BY_IP_ADDRESS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderString(resource.createdByIpAddress)
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.EXPIRES_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderDate(resource.expiresAt)
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.LAST_USED_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderDate(resource.lastUsedAt)
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.LAST_USED_BY_IP_ADDRESS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderString(resource.lastUsedByIpAddress)
    },
    {
        name: ApiClientAuthorizationPanelColumnNames.SCOPES,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ApiClientAuthorization) => renderString(resource.scopes.join(', '))
    },
];

const DEFAULT_MESSAGE = 'Your api client authorization list is empty.';

export interface ApiClientAuthorizationPanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export interface ApiClientAuthorizationPanelRootDataProps {
    resources: ResourcesState;
}

type ApiClientAuthorizationPanelRootProps = ApiClientAuthorizationPanelRootActionProps
    & ApiClientAuthorizationPanelRootDataProps & WithStyles<CssRules>;

export const ApiClientAuthorizationPanelRoot = withStyles(styles)(
    ({ classes, onItemDoubleClick, onItemClick, onContextMenu }: ApiClientAuthorizationPanelRootProps) =>
        <div className={classes.root}><DataExplorer
            id={API_CLIENT_AUTHORIZATION_PANEL_ID}
            onRowClick={onItemClick}
            onRowDoubleClick={onItemDoubleClick}
            onContextMenu={onContextMenu}
            contextMenuColumn={true}
            hideColumnSelector
            hideSearchInput
            defaultViewIcon={ShareMeIcon}
            defaultViewMessages={[DEFAULT_MESSAGE]} />
        </div>
);
