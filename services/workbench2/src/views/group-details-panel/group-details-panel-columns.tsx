// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns } from 'components/data-table/data-column';
import { PermissionResource } from 'models/permission';
import { createTree } from 'models/tree';
import {
    ResourceLinkHeadUuid,
    ResourceLinkTailUsername,
    ResourceLinkHeadPermissionLevel,
    ResourceLinkTailPermissionLevel,
    ResourceLinkHead,
    ResourceLinkTail,
    ResourceLinkDelete,
    ResourcePermissionsDelete,
    ResourceLinkTailAccountStatus,
    ResourceLinkTailIsVisible,
} from 'views-components/data-explorer/renderers';

export enum GroupDetailsPanelMembersColumnNames {
    FULL_NAME = "Name",
    USERNAME = "Username",
    STATUS = "Account Status",
    VISIBLE = "Visible to other members",
    PERMISSION = "Permission",
    REMOVE = "Remove",
}

export enum GroupDetailsPanelPermissionsColumnNames {
    NAME = "Name",
    PERMISSION = "Permission",
    UUID = "UUID",
    REMOVE = "Remove",
}

export const groupDetailsMembersPanelColumns: DataColumns<string, PermissionResource> = [
    {
        name: GroupDetailsPanelMembersColumnNames.FULL_NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTail uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.USERNAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailUsername uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailAccountStatus uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.VISIBLE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailIsVisible uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.PERMISSION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailPermissionLevel uuid={uuid} />
    },
    {
        name: GroupDetailsPanelMembersColumnNames.REMOVE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkDelete uuid={uuid} />
    },
];

export const groupDetailsPermissionsPanelColumns: DataColumns<string, PermissionResource> = [
    {
        name: GroupDetailsPanelPermissionsColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: GroupDetailsPanelPermissionsColumnNames.PERMISSION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadPermissionLevel uuid={uuid} />
    },
    {
        name: GroupDetailsPanelPermissionsColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadUuid uuid={uuid} />
    },
    {
        name: GroupDetailsPanelPermissionsColumnNames.REMOVE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourcePermissionsDelete uuid={uuid} />
    },
];
