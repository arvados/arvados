// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns } from 'components/data-table/data-column';
import { ResourceLinkHeadUuid, ResourceLinkHeadPermissionLevel, ResourceLinkHead, ResourceLinkDelete, ResourceLinkTailIsVisible } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { PermissionResource } from 'models/permission';


export enum UserProfileGroupsColumnNames {
    NAME = "Name",
    PERMISSION = "Permission",
    VISIBLE = "Visible to other members",
    UUID = "UUID",
    REMOVE = "Remove"
}

export const userProfileGroupsColumns: DataColumns<string, PermissionResource> = [
    {
        name: UserProfileGroupsColumnNames.NAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.PERMISSION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadPermissionLevel uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.VISIBLE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTailIsVisible uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHeadUuid uuid={uuid} />
    },
    {
        name: UserProfileGroupsColumnNames.REMOVE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkDelete uuid={uuid} />
    },
];
