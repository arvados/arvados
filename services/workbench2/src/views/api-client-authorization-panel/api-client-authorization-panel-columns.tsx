// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { createTree } from 'models/tree';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import {
    CommonUuid, TokenApiToken, TokenCreatedByIpAddress, TokenExpiresAt,
    TokenLastUsedAt, TokenLastUsedByIpAddress, TokenScopes, TokenUserId
} from 'views-components/data-explorer/renderers';
import { ApiClientAuthorization } from 'models/api-client-authorization';

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

export const apiClientAuthorizationPanelColumns: DataColumns<string, ApiClientAuthorization> = [
    {
        name: ApiClientAuthorizationPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "uuid" },
        filters: createTree(),
        render: uuid => <CommonUuid uuid={uuid} />
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
